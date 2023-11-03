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

create index chair_stock_price_id_index
    on isuumo.chair (stock, price, id);

create index estate_rent_id_index
    on isuumo.estate (rent, id);

ALTER TABLE isuumo.chair ADD COLUMN features_array text[] GENERATED ALWAYS AS (regexp_split_to_array(features, ',')) STORED;

CREATE INDEX idx_features_array ON chair USING gin(features_array);

ALTER TABLE isuumo.chair
ADD COLUMN price_range int GENERATED ALWAYS AS (CASE WHEN price < 3000 THEN 0 WHEN 3000 <= price and price < 6000 THEN 1 WHEN 6000 <= price and price < 9000 THEN 2 WHEN 9000 <= price and price < 12000 THEN 3 WHEN 12000 <= price and price < 15000 THEN 4 WHEN 15000 <= price THEN 5 END) STORED;

create index chair_price_range_popularity_id_index
    on isuumo.chair (price_range asc, popularity desc, id asc);
