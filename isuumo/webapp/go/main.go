package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

//go:generate go run github.com/mackee/go-sqlla/v2/cmd/sqlla

const (
	Limit        = 20
	NazotteLimit = 50
)

var (
	chairDB               *sqlx.DB
	estateDB              *sqlx.DB
	rdb                   *redis.Client
	chairSearchCondition  ChairSearchCondition
	estateSearchCondition EstateSearchCondition
)

type InitializeResponse struct {
	Language string `json:"language"`
}

//sqlla:table chair
type Chair struct {
	ID            int64  `db:"id" json:"id"`
	Name          string `db:"name" json:"name"`
	Description   string `db:"description" json:"description"`
	Thumbnail     string `db:"thumbnail" json:"thumbnail"`
	Price         int64  `db:"price" json:"price"`
	Height        int64  `db:"height" json:"height"`
	Width         int64  `db:"width" json:"width"`
	Depth         int64  `db:"depth" json:"depth"`
	Color         string `db:"color" json:"color"`
	Features      string `db:"features" json:"features"`
	Kind          string `db:"kind" json:"kind"`
	Popularity    int64  `db:"popularity" json:"-"`
	Stock         int64  `db:"stock" json:"-"`
	FeaturesArray string `db:"features_array" json:"-"`
	PriceRange    int64  `db:"price_range" json:"-"`
	HeightRange   int64  `db:"height_range" json:"-"`
	WidthRange    int64  `db:"width_range" json:"-"`
	DepthRange    int64  `db:"depth_range" json:"-"`
}

type ChairSearchResponse struct {
	Count  int64   `json:"count"`
	Chairs []Chair `json:"chairs"`
}

type ChairListResponse struct {
	Chairs []Chair `json:"chairs"`
}

// Estate 物件
//
//sqlla:table estate
type Estate struct {
	ID              int64   `db:"id" json:"id"`
	Thumbnail       string  `db:"thumbnail" json:"thumbnail"`
	Name            string  `db:"name" json:"name"`
	Description     string  `db:"description" json:"description"`
	Latitude        float64 `db:"latitude" json:"latitude"`
	Longitude       float64 `db:"longitude" json:"longitude"`
	Address         string  `db:"address" json:"address"`
	Rent            int64   `db:"rent" json:"rent"`
	DoorHeight      int64   `db:"door_height" json:"doorHeight"`
	DoorWidth       int64   `db:"door_width" json:"doorWidth"`
	Features        string  `db:"features" json:"features"`
	Popularity      int64   `db:"popularity" json:"-"`
	FeaturesArray   string  `db:"features_array" json:"-"`
	RentRange       int64   `db:"rent_range" json:"-"`
	DoorHeightRange int64   `db:"door_height_range" json:"-"`
	DoorWidthRange  int64   `db:"door_width_range" json:"-"`
}

// EstateSearchResponse estate/searchへのレスポンスの形式
type EstateSearchResponse struct {
	Count   int64    `json:"count"`
	Estates []Estate `json:"estates"`
}

type EstateListResponse struct {
	Estates []Estate `json:"estates"`
}

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Coordinates struct {
	Coordinates []Coordinate `json:"coordinates"`
}

type Range struct {
	ID  int64 `json:"id"`
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

type RangeCondition struct {
	Prefix string   `json:"prefix"`
	Suffix string   `json:"suffix"`
	Ranges []*Range `json:"ranges"`
}

type ListCondition struct {
	List []string `json:"list"`
}

type EstateSearchCondition struct {
	DoorWidth  RangeCondition `json:"doorWidth"`
	DoorHeight RangeCondition `json:"doorHeight"`
	Rent       RangeCondition `json:"rent"`
	Feature    ListCondition  `json:"feature"`
}

type ChairSearchCondition struct {
	Width   RangeCondition `json:"width"`
	Height  RangeCondition `json:"height"`
	Depth   RangeCondition `json:"depth"`
	Price   RangeCondition `json:"price"`
	Color   ListCondition  `json:"color"`
	Feature ListCondition  `json:"feature"`
	Kind    ListCondition  `json:"kind"`
}

type BoundingBox struct {
	// TopLeftCorner 緯度経度が共に最小値になるような点の情報を持っている
	TopLeftCorner Coordinate
	// BottomRightCorner 緯度経度が共に最大値になるような点の情報を持っている
	BottomRightCorner Coordinate
}

type PostgresEnv struct {
	Host     string
	Port     string
	User     string
	DBName   string
	Password string
}

type RecordMapper struct {
	Record []string

	offset int
	err    error
}

func (r *RecordMapper) next() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	if r.offset >= len(r.Record) {
		r.err = fmt.Errorf("too many read")
		return "", r.err
	}
	s := r.Record[r.offset]
	r.offset++
	return s, nil
}

func (r *RecordMapper) NextInt() int {
	s, err := r.next()
	if err != nil {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		r.err = err
		return 0
	}
	return i
}

func (r *RecordMapper) NextFloat() float64 {
	s, err := r.next()
	if err != nil {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		r.err = err
		return 0
	}
	return f
}

func (r *RecordMapper) NextString() string {
	s, err := r.next()
	if err != nil {
		return ""
	}
	return s
}

func (r *RecordMapper) Err() error {
	return r.err
}

func NewPostgresEnv() *PostgresEnv {
	return &PostgresEnv{
		Host:     getEnv("DB_HOSTNAME", "127.0.0.1"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "isucon"),
		DBName:   getEnv("DB_DBNAME", "isuumo"),
		Password: getEnv("DB_PASS", "isucon"),
	}
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultValue
}

// ConnectDB isuumoデータベースに接続する
func (mc *PostgresEnv) ConnectDB() (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v", mc.User, mc.Password, mc.Host, mc.Port, mc.DBName)
	return sqlx.Open("mysql", dsn)
}

func init() {
	jsonText, err := os.ReadFile("../fixture/chair_condition.json")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	json.Unmarshal(jsonText, &chairSearchCondition)

	jsonText, err = os.ReadFile("../fixture/estate_condition.json")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	json.Unmarshal(jsonText, &estateSearchCondition)
}

func main() {
	tp, _ := initTracer(context.Background())
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	// Echo instance
	e := echo.New()
	e.Debug = true
	e.Logger.SetLevel(log.DEBUG)

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(otelecho.Middleware("isuumo"))

	// Initialize
	e.POST("/initialize", initialize)

	// Chair Handler
	e.GET("/api/chair/:id", getChairDetail)
	e.POST("/api/chair", postChair)
	e.GET("/api/chair/search", searchChairs)
	e.GET("/api/chair/low_priced", getLowPricedChair)
	e.GET("/api/chair/search/condition", getChairSearchCondition)
	e.POST("/api/chair/buy/:id", buyChair)

	// Estate Handler
	e.GET("/api/estate/:id", getEstateDetail)
	e.POST("/api/estate", postEstate)
	e.GET("/api/estate/search", searchEstates)
	e.GET("/api/estate/low_priced", getLowPricedEstate)
	e.POST("/api/estate/req_doc/:id", postEstateRequestDocument)
	e.POST("/api/estate/nazotte", searchEstateNazotte)
	e.GET("/api/estate/search/condition", getEstateSearchCondition)
	e.GET("/api/recommended_estate/:id", searchRecommendedEstateWithChair)

	var err error
	estateDB, err = GetDB(GetEnv("DB_HOSTNAME1", "192.168.0.12"))
	if err != nil {
		e.Logger.Fatalf("DB connection failed : %v", err)
	}
	estateDB.SetMaxOpenConns(10)
	defer estateDB.Close()

	chairDB, err = GetDB(GetEnv("DB_HOSTNAME2", "192.168.0.13"))
	if err != nil {
		e.Logger.Fatalf("DB connection failed : %v", err)
	}
	chairDB.SetMaxOpenConns(10)
	defer chairDB.Close()

	rdb = redis.NewClient(&redis.Options{
		Addr:     GetEnv("REDIS_HOSTNAME", "127.0.0.1") + ":6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	if err = rdb.Ping(context.Background()).Err(); err != nil {
		e.Logger.Fatalf("Redis connection failed : %v", err)
	}

	// Start server
	serverPort := fmt.Sprintf(":%v", getEnv("SERVER_PORT", "1323"))
	e.Logger.Fatal(e.Start(serverPort))
}

func initialize(c echo.Context) error {
	sqlDir := filepath.Join("..", "mysql", "db")
	paths := []string{
		filepath.Join(sqlDir, "0_Schema.sql"),
		filepath.Join(sqlDir, "1_DummyEstateData.sql"),
		filepath.Join(sqlDir, "2_DummyChairData.sql"),
	}

	pgConnectionData := NewPostgresEnv()
	for _, p := range paths {
		sqlFile, _ := filepath.Abs(p)
		cmdStr := fmt.Sprintf("psql -h %v -U %v -d %v -f %v",
			pgConnectionData.Host,
			pgConnectionData.User,
			pgConnectionData.DBName,
			sqlFile,
		)
		cmd := exec.Command("bash", "-c", cmdStr)
		cmd.Env = append(cmd.Env, "PGPASSWORD="+pgConnectionData.Password)
		if err := cmd.Run(); err != nil {
			c.Logger().Errorf("Initialize script error : %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	// 在庫0の修正
	if err := rdb.FlushAll(c.Request().Context()).Err(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	return c.JSON(http.StatusOK, InitializeResponse{
		Language: "go",
	})
}

func getChairDetail(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Echo().Logger.Errorf("Request parameter \"id\" parse error : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	ctx := c.Request().Context()
	chair := Chair{}
	query := `SELECT * FROM chair WHERE id = ?`
	err = chairDB.GetContext(ctx, &chair, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.Echo().Logger.Infof("requested id's chair not found : %v", id)
			return c.NoContent(http.StatusNotFound)
		}
		c.Echo().Logger.Errorf("Failed to get the chair from id : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	} else if chair.Stock <= 0 {
		c.Echo().Logger.Infof("requested id's chair is sold out : %v", id)
		return c.NoContent(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, chair)
}

func postChair(c echo.Context) error {
	header, err := c.FormFile("chairs")
	if err != nil {
		c.Logger().Errorf("failed to get form file: %v", err)
		return c.NoContent(http.StatusBadRequest)
	}
	f, err := header.Open()
	if err != nil {
		c.Logger().Errorf("failed to open form file: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		c.Logger().Errorf("failed to read csv: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	ctx := c.Request().Context()
	bi := NewChairSQL().BulkInsert()
	for _, row := range records {
		rm := RecordMapper{Record: row}
		id := rm.NextInt()
		name := rm.NextString()
		description := rm.NextString()
		thumbnail := rm.NextString()
		price := rm.NextInt()
		height := rm.NextInt()
		width := rm.NextInt()
		depth := rm.NextInt()
		color := rm.NextString()
		features := rm.NextString()
		kind := rm.NextString()
		popularity := rm.NextInt()
		stock := rm.NextInt()
		if err := rm.Err(); err != nil {
			c.Logger().Errorf("failed to read record: %v", err)
			return c.NoContent(http.StatusBadRequest)
		}
		bi.Append(
			NewChairSQL().Insert().
				ValueID(int64(id)).
				ValueName(name).
				ValueDescription(description).
				ValueThumbnail(thumbnail).
				ValuePrice(int64(price)).
				ValueHeight(int64(height)).
				ValueWidth(int64(width)).
				ValueDepth(int64(depth)).
				ValueColor(color).
				ValueFeatures(features).
				ValueKind(kind).
				ValuePopularity(int64(popularity)).
				ValueStock(int64(stock)),
		)
	}
	if _, err := bi.ExecContext(ctx, chairDB); err != nil {
		c.Logger().Errorf("failed to insert chair: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusCreated)
}

func searchChairs(c echo.Context) error {
	conditions := make([]string, 0)
	params := make([]interface{}, 0)

	if c.QueryParam("priceRangeId") != "" {
		conditions = append(conditions, "price_range = ?")
		params = append(params, c.QueryParam("priceRangeId"))
	}

	if c.QueryParam("heightRangeId") != "" {
		conditions = append(conditions, "height_range = ?")
		params = append(params, c.QueryParam("heightRangeId"))
	}

	if c.QueryParam("widthRangeId") != "" {
		conditions = append(conditions, "width_range = ?")
		params = append(params, c.QueryParam("widthRangeId"))
	}

	if c.QueryParam("depthRangeId") != "" {
		conditions = append(conditions, "depth_range = ?")
		params = append(params, c.QueryParam("depthRangeId"))
	}

	if c.QueryParam("kind") != "" {
		conditions = append(conditions, "kind = ?")
		params = append(params, c.QueryParam("kind"))
	}

	if c.QueryParam("color") != "" {
		conditions = append(conditions, "color = ?")
		params = append(params, c.QueryParam("color"))
	}

	if c.QueryParam("features") != "" {
		ss := strings.Split(c.QueryParam("features"), ",")
		if len(ss) > 0 {
			for _, s := range ss {
				params = append(params, s)
			}
			conditions = append(
				conditions,
				fmt.Sprintf("features_array @> ARRAY[?%s]",
					strings.Repeat(",?", len(ss)-1)),
			)
		}
	}

	if len(conditions) == 0 {
		c.Echo().Logger.Infof("Search condition not found")
		return c.NoContent(http.StatusBadRequest)
	}

	conditions = append(conditions, "stock > 0")

	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil {
		c.Logger().Infof("Invalid format page parameter : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	perPage, err := strconv.Atoi(c.QueryParam("perPage"))
	if err != nil {
		c.Logger().Infof("Invalid format perPage parameter : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	searchQuery := "SELECT * FROM chair WHERE "
	countQuery := "SELECT COUNT(*) FROM chair WHERE "
	searchCondition := strings.Join(conditions, " AND ")
	limitOffset := " ORDER BY popularity DESC, id ASC LIMIT ? OFFSET ?"

	ctx := c.Request().Context()
	var res ChairSearchResponse
	err = chairDB.GetContext(ctx, &res.Count, countQuery+searchCondition, params...)
	if err != nil {
		c.Logger().Errorf("searchChairs DB execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	chairs := []Chair{}
	params = append(params, perPage, page*perPage)
	err = chairDB.SelectContext(ctx, &chairs, searchQuery+searchCondition+limitOffset, params...)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusOK, ChairSearchResponse{Count: 0, Chairs: []Chair{}})
		}
		c.Logger().Errorf("searchChairs DB execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res.Chairs = chairs

	return c.JSON(http.StatusOK, res)
}

const soldOutChairKey = "sold_out_chair"

func buyChair(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		c.Echo().Logger.Infof("post buy chair failed : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	_, ok := m["email"].(string)
	if !ok {
		c.Echo().Logger.Info("post buy chair failed : email not found in request body")
		return c.NoContent(http.StatusBadRequest)
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Echo().Logger.Infof("post buy chair failed : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	ctx := c.Request().Context()

	var stock int64
	row := chairDB.QueryRowContext(ctx, "UPDATE chair SET stock = stock - 1 WHERE id = ? AND stock > 0 RETURNING stock", id)
	if err := row.Scan(&stock); err != nil {
		if err == sql.ErrNoRows {
			c.Echo().Logger.Infof("buyChair chair id \"%v\" not found", id)
			return c.NoContent(http.StatusNotFound)
		}
		c.Echo().Logger.Errorf("chair stock update failed : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// 残り1つを購入したことになるので在庫切れリストに追加する
	if stock == 0 {
		if err := rdb.SAdd(c.Request().Context(), soldOutChairKey, id).Err(); err != nil {
			c.Echo().Logger.Errorf("failed to insert sold_out_chair to redis, id: %v", id)
			return c.NoContent(http.StatusInsufficientStorage)
		}
	}

	return c.NoContent(http.StatusOK)
}

func getChairSearchCondition(c echo.Context) error {
	return c.JSON(http.StatusOK, chairSearchCondition)
}

func getLowPricedChair(c echo.Context) error {
	ctx := c.Request().Context()
	var chairs []Chair
	query := `SELECT * FROM chair WHERE stock > 0 ORDER BY price ASC, id ASC LIMIT ?`
	err := chairDB.SelectContext(ctx, &chairs, query, Limit)
	if err != nil {
		if err == sql.ErrNoRows {
			c.Logger().Error("getLowPricedChair not found")
			return c.JSON(http.StatusOK, ChairListResponse{[]Chair{}})
		}
		c.Logger().Errorf("getLowPricedChair DB execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ChairListResponse{Chairs: chairs})
}

func getEstateDetail(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Echo().Logger.Infof("Request parameter \"id\" parse error : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	ctx := c.Request().Context()
	var estate Estate
	err = estateDB.GetContext(ctx, &estate, "SELECT * FROM estate WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.Echo().Logger.Infof("getEstateDetail estate id %v not found", id)
			return c.NoContent(http.StatusNotFound)
		}
		c.Echo().Logger.Errorf("Database Execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, estate)
}

func getRange(cond RangeCondition, rangeID string) (*Range, error) {
	RangeIndex, err := strconv.Atoi(rangeID)
	if err != nil {
		return nil, err
	}

	if RangeIndex < 0 || len(cond.Ranges) <= RangeIndex {
		return nil, fmt.Errorf("Unexpected Range ID")
	}

	return cond.Ranges[RangeIndex], nil
}

func postEstate(c echo.Context) error {
	header, err := c.FormFile("estates")
	if err != nil {
		c.Logger().Errorf("failed to get form file: %v", err)
		return c.NoContent(http.StatusBadRequest)
	}
	f, err := header.Open()
	if err != nil {
		c.Logger().Errorf("failed to open form file: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		c.Logger().Errorf("failed to read csv: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	ctx := c.Request().Context()
	bi := NewEstateSQL().BulkInsert()
	for _, row := range records {
		rm := RecordMapper{Record: row}
		id := rm.NextInt()
		name := rm.NextString()
		description := rm.NextString()
		thumbnail := rm.NextString()
		address := rm.NextString()
		latitude := rm.NextFloat()
		longitude := rm.NextFloat()
		rent := rm.NextInt()
		doorHeight := rm.NextInt()
		doorWidth := rm.NextInt()
		features := rm.NextString()
		popularity := rm.NextInt()
		if err := rm.Err(); err != nil {
			c.Logger().Errorf("failed to read record: %v", err)
			return c.NoContent(http.StatusBadRequest)
		}
		bi.Append(
			NewEstateSQL().Insert().
				ValueID(int64(id)).
				ValueThumbnail(thumbnail).
				ValueName(name).
				ValueDescription(description).
				ValueLatitude(latitude).
				ValueLongitude(longitude).
				ValueAddress(address).
				ValueRent(int64(rent)).
				ValueDoorHeight(int64(doorHeight)).
				ValueDoorWidth(int64(doorWidth)).
				ValueFeatures(features).
				ValuePopularity(int64(popularity)),
		)
	}
	if _, err := bi.ExecContext(ctx, estateDB); err != nil {
		c.Logger().Errorf("failed to insert estate: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusCreated)
}

func searchEstates(c echo.Context) error {
	conditions := make([]string, 0)
	params := make([]interface{}, 0)

	if c.QueryParam("doorHeightRangeId") != "" {
		conditions = append(conditions, "door_height_range = ?")
		params = append(params, c.QueryParam("doorHeightRangeId"))
	}

	c.Echo().Logger.Debug("request uri: ", c.Request().RequestURI)
	c.Echo().Logger.Debug("doorWidthRangeId: ", c.QueryParam("doorWidthRangeId"))
	if c.QueryParam("doorWidthRangeId") != "" {
		conditions = append(conditions, "door_width_range = ?")
		params = append(params, c.QueryParam("doorWidthRangeId"))
	}

	if c.QueryParam("rentRangeId") != "" {
		conditions = append(conditions, "rent_range = ?")
		params = append(params, c.QueryParam("rentRangeId"))
	}

	if c.QueryParam("features") != "" {
		ss := strings.Split(c.QueryParam("features"), ",")
		if len(ss) > 0 {
			for _, s := range ss {
				params = append(params, s)
			}
			conditions = append(
				conditions,
				fmt.Sprintf("features_array @> ARRAY[?%s]",
					strings.Repeat(",?", len(ss)-1)),
			)
		}
	}

	if len(conditions) == 0 {
		c.Echo().Logger.Infof("searchEstates search condition not found")
		return c.NoContent(http.StatusBadRequest)
	}

	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil {
		c.Logger().Infof("Invalid format page parameter : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	perPage, err := strconv.Atoi(c.QueryParam("perPage"))
	if err != nil {
		c.Logger().Infof("Invalid format perPage parameter : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	searchQuery := "SELECT * FROM estate WHERE "
	countQuery := "SELECT COUNT(*) FROM estate WHERE "
	searchCondition := strings.Join(conditions, " AND ")
	limitOffset := " ORDER BY popularity DESC, id ASC LIMIT ? OFFSET ?"

	ctx := c.Request().Context()
	var res EstateSearchResponse
	err = estateDB.GetContext(ctx, &res.Count, countQuery+searchCondition, params...)
	if err != nil {
		c.Logger().Errorf("searchEstates DB execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	estates := []Estate{}
	params = append(params, perPage, page*perPage)
	err = estateDB.SelectContext(ctx, &estates, searchQuery+searchCondition+limitOffset, params...)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusOK, EstateSearchResponse{Count: 0, Estates: []Estate{}})
		}
		c.Logger().Errorf("searchEstates DB execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res.Estates = estates

	return c.JSON(http.StatusOK, res)
}

func getLowPricedEstate(c echo.Context) error {
	ctx := c.Request().Context()
	estates := make([]Estate, 0, Limit)
	query := `SELECT * FROM estate ORDER BY rent ASC, id ASC LIMIT ?`
	err := estateDB.SelectContext(ctx, &estates, query, Limit)
	if err != nil {
		if err == sql.ErrNoRows {
			c.Logger().Error("getLowPricedEstate not found")
			return c.JSON(http.StatusOK, EstateListResponse{[]Estate{}})
		}
		c.Logger().Errorf("getLowPricedEstate DB execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, EstateListResponse{Estates: estates})
}

func searchRecommendedEstateWithChair(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Logger().Infof("Invalid format searchRecommendedEstateWithChair id : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	ctx := c.Request().Context()
	chair := Chair{}
	query := `SELECT * FROM chair WHERE id = ?`
	err = chairDB.GetContext(ctx, &chair, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.Logger().Infof("Requested chair id \"%v\" not found", id)
			return c.NoContent(http.StatusBadRequest)
		}
		c.Logger().Errorf("Database execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var estates []Estate
	sorted := []int64{chair.Width, chair.Depth, chair.Height}
	slices.Sort(sorted)
	l1, l2 := sorted[0], sorted[1]
	query = `SELECT * from (select * from estate where door_width >= ? AND door_height >= ? ORDER BY popularity DESC ,id limit ?) as t
union
SELECT * from  (select * from estate where door_width >= ? AND door_height >= ? ORDER BY popularity DESC ,id limit ?) as t2 ORDER BY popularity DESC ,id limit ?;`
	err = estateDB.SelectContext(ctx, &estates, query, l1, l2, Limit, l2, l1, Limit, Limit)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusOK, EstateListResponse{[]Estate{}})
		}
		c.Logger().Errorf("Database execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, EstateListResponse{Estates: estates})
}

func searchEstateNazotte(c echo.Context) error {
	coordinates := Coordinates{}
	err := c.Bind(&coordinates)
	if err != nil {
		c.Echo().Logger.Infof("post search estate nazotte failed : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	if len(coordinates.Coordinates) == 0 {
		return c.NoContent(http.StatusBadRequest)
	}

	ctx := c.Request().Context()
	b := coordinates.getBoundingBox()
	estatesInBoundingBox := []Estate{}
	query := `SELECT * FROM estate WHERE latitude <= ? AND latitude >= ? AND longitude <= ? AND longitude >= ? ORDER BY popularity DESC, id ASC`
	err = estateDB.SelectContext(ctx, &estatesInBoundingBox, query, b.BottomRightCorner.Latitude, b.TopLeftCorner.Latitude, b.BottomRightCorner.Longitude, b.TopLeftCorner.Longitude)
	if err == sql.ErrNoRows {
		c.Echo().Logger.Infof("select * from estate where latitude ...", err)
		return c.JSON(http.StatusOK, EstateSearchResponse{Count: 0, Estates: []Estate{}})
	} else if err != nil {
		c.Echo().Logger.Errorf("database execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	estatesInPolygon := []Estate{}
	ring := coordinates.ring()
	for _, estate := range estatesInBoundingBox {
		if !planar.RingContains(ring, orb.Point{estate.Latitude, estate.Longitude}) {
			continue
		} else {
			estatesInPolygon = append(estatesInPolygon, estate)
		}
	}

	var re EstateSearchResponse
	re.Estates = []Estate{}
	if len(estatesInPolygon) > NazotteLimit {
		re.Estates = estatesInPolygon[:NazotteLimit]
	} else {
		re.Estates = estatesInPolygon
	}
	re.Count = int64(len(re.Estates))

	return c.JSON(http.StatusOK, re)
}

func postEstateRequestDocument(c echo.Context) error {
	m := echo.Map{}
	if err := c.Bind(&m); err != nil {
		c.Echo().Logger.Infof("post request document failed : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	_, ok := m["email"].(string)
	if !ok {
		c.Echo().Logger.Info("post request document failed : email not found in request body")
		return c.NoContent(http.StatusBadRequest)
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Echo().Logger.Infof("post request document failed : %v", err)
		return c.NoContent(http.StatusBadRequest)
	}

	ctx := c.Request().Context()
	estate := Estate{}
	query := `SELECT * FROM estate WHERE id = ?`
	err = estateDB.GetContext(ctx, &estate, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.NoContent(http.StatusNotFound)
		}
		c.Logger().Errorf("postEstateRequestDocument DB execution error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func getEstateSearchCondition(c echo.Context) error {
	return c.JSON(http.StatusOK, estateSearchCondition)
}

func (cs Coordinates) getBoundingBox() BoundingBox {
	coordinates := cs.Coordinates
	boundingBox := BoundingBox{
		TopLeftCorner: Coordinate{
			Latitude: coordinates[0].Latitude, Longitude: coordinates[0].Longitude,
		},
		BottomRightCorner: Coordinate{
			Latitude: coordinates[0].Latitude, Longitude: coordinates[0].Longitude,
		},
	}
	for _, coordinate := range coordinates {
		if boundingBox.TopLeftCorner.Latitude > coordinate.Latitude {
			boundingBox.TopLeftCorner.Latitude = coordinate.Latitude
		}
		if boundingBox.TopLeftCorner.Longitude > coordinate.Longitude {
			boundingBox.TopLeftCorner.Longitude = coordinate.Longitude
		}

		if boundingBox.BottomRightCorner.Latitude < coordinate.Latitude {
			boundingBox.BottomRightCorner.Latitude = coordinate.Latitude
		}
		if boundingBox.BottomRightCorner.Longitude < coordinate.Longitude {
			boundingBox.BottomRightCorner.Longitude = coordinate.Longitude
		}
	}
	return boundingBox
}

func (cs Coordinates) ring() orb.Ring {
	ring := make(orb.Ring, 0, len(cs.Coordinates))
	for _, c := range cs.Coordinates {
		ring = append(ring, orb.Point{c.Latitude, c.Longitude})
	}
	return ring
}

func (cs Coordinates) coordinatesToText() string {
	points := make([]string, 0, len(cs.Coordinates))
	for _, c := range cs.Coordinates {
		points = append(points, fmt.Sprintf("%f %f", c.Latitude, c.Longitude))
	}
	return fmt.Sprintf("'POLYGON((%s))'", strings.Join(points, ","))
}
