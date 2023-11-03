DROP DATABASE IF EXISTS isuumo;
CREATE DATABASE isuumo;

DROP TABLE IF EXISTS isuumo.estate;
DROP TABLE IF EXISTS isuumo.chair;

CREATE TABLE isuumo.estate
(
    id          INTEGER             NOT NULL PRIMARY KEY,
    name        VARCHAR(64)         NOT NULL,
    description VARCHAR(4096)       NOT NULL,
    thumbnail   VARCHAR(128)        NOT NULL,
    address     VARCHAR(128)        NOT NULL,
    latitude    DOUBLE PRECISION    NOT NULL,
    longitude   DOUBLE PRECISION    NOT NULL,
    rent        INTEGER             NOT NULL,
    door_height INTEGER             NOT NULL,
    door_width  INTEGER             NOT NULL,
    features    VARCHAR(64)         NOT NULL,
    popularity  INTEGER             NOT NULL
);

CREATE TABLE isuumo.chair
(
    id          INTEGER         NOT NULL PRIMARY KEY,
    name        VARCHAR(64)     NOT NULL,
    description VARCHAR(4096)   NOT NULL,
    thumbnail   VARCHAR(128)    NOT NULL,
    price       INTEGER         NOT NULL,
    height      INTEGER         NOT NULL,
    width       INTEGER         NOT NULL,
    depth       INTEGER         NOT NULL,
    color       VARCHAR(64)     NOT NULL,
    features    VARCHAR(64)     NOT NULL,
    kind        VARCHAR(64)     NOT NULL,
    popularity  INTEGER         NOT NULL,
    stock       INTEGER         NOT NULL
);


create index estate_popularity_id_index
    on isuumo.estate (popularity desc, id asc);

create index estate_rent_popularity_id_index
    on isuumo.estate (rent, popularity desc, id asc);

create index chair_price_popularity_id_index
    on isuumo.chair (price, popularity desc, id asc);

create index chair_popularity_id_index
    on isuumo.chair (popularity desc, id asc);

create index chair_stock_price_id_index
    on isuumo.chair (stock, price, id);


-- chair
ALTER TABLE isuumo.chair ADD COLUMN features_array text[] GENERATED ALWAYS AS (regexp_split_to_array(features, ',')) STORED;

CREATE INDEX idx_features_array ON chair USING gin(features_array);

ALTER TABLE isuumo.chair
ADD COLUMN price_range int GENERATED ALWAYS AS (CASE WHEN price < 3000 THEN 0 WHEN 3000 <= price and price < 6000 THEN 1 WHEN 6000 <= price and price < 9000 THEN 2 WHEN 9000 <= price and price < 12000 THEN 3 WHEN 12000 <= price and price < 15000 THEN 4 WHEN 15000 <= price THEN 5 END) STORED;

create index chair_price_range_popularity_id_index
    on isuumo.chair (price_range asc, popularity desc, id asc);

-- 80cm未満: 0, 80cm以上110cm未満: 1, 110cm以上150cm未満: 2, 150cm以上: 3
ALTER TABLE isuumo.chair
ADD COLUMN height_range int GENERATED ALWAYS AS (CASE WHEN height < 80 THEN 0 WHEN 80 <= height and height < 110 THEN 1 WHEN 110 <= height and height < 150 THEN 2 WHEN 150 <= height THEN 3 END) STORED;

create index chair_height_range_popularity_id_index
    on isuumo.chair (height_range asc, popularity desc, id asc);

ALTER TABLE isuumo.chair
ADD COLUMN width_range int GENERATED ALWAYS AS (CASE WHEN width < 80 THEN 0 WHEN 80 <= width and width < 110 THEN 1 WHEN 110 <= width and width < 150 THEN 2 WHEN 150 <= width THEN 3 END) STORED;

create index chair_width_range_popularity_id_index
    on isuumo.chair (width_range asc, popularity desc, id asc);

ALTER TABLE isuumo.chair
ADD COLUMN depth_range int GENERATED ALWAYS AS (CASE WHEN depth < 80 THEN 0 WHEN 80 <= depth and depth < 110 THEN 1 WHEN 110 <= depth and depth < 150 THEN 2 WHEN 150 <= depth THEN 3 END) STORED;

create index chair_depth_range_popularity_id_index
    on isuumo.chair (depth_range asc, popularity desc, id asc);

-- estate
ALTER TABLE isuumo.estate ADD COLUMN features_array text[] GENERATED ALWAYS AS (regexp_split_to_array(features, ',')) STORED;
CREATE INDEX idx_estate_features_array ON estate USING gin(features_array);

ALTER TABLE isuumo.estate
ADD COLUMN rent_range int GENERATED ALWAYS AS (CASE WHEN rent < 50000 THEN 0 WHEN 50000 <= rent and rent < 100000 THEN 1 WHEN 100000 <= rent and rent < 150000 THEN 2 WHEN 150000 <= rent THEN 3 END) STORED;

create index estate_rent_range_popularity_id_index
    on isuumo.estate (rent_range asc, popularity desc, id asc);

-- 80cm未満: 0, 80cm以上110cm未満: 1, 110cm以上150cm未満: 2, 150cm以上: 3
ALTER TABLE isuumo.estate
ADD COLUMN door_height_range int GENERATED ALWAYS AS (CASE WHEN door_height < 80 THEN 0 WHEN 80 <= door_height and door_height < 110 THEN 1 WHEN 110 <= door_height and door_height < 150 THEN 2 WHEN 150 <= door_height THEN 3 END) STORED;

create index estate_door_height_range_popularity_id_index
    on isuumo.estate (door_height_range asc, popularity desc, id asc);

ALTER TABLE isuumo.estate
ADD COLUMN door_width_range int GENERATED ALWAYS AS (CASE WHEN door_width < 80 THEN 0 WHEN 80 <= door_width and door_width < 110 THEN 1 WHEN 110 <= door_width and door_width < 150 THEN 2 WHEN 150 <= door_width THEN 3 END) STORED;

create index estate_door_width_range_popularity_id_index
    on isuumo.estate (door_width_range asc, popularity desc, id asc);