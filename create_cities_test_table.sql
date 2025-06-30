-- 创建测试表 cities_test
CREATE TABLE IF NOT EXISTS cities_test
(
    id           mediumint unsigned auto_increment
        primary key,
    name         varchar(255)       default ''                    not null,
    state_id     mediumint unsigned default '0'                   not null,
    state_code   varchar(255)       default ''                    not null,
    country_id   mediumint unsigned default '0'                   not null,
    country_code char(2)            default ''                    not null,
    latitude     decimal(10, 8)     default 0.00000000            not null,
    longitude    decimal(11, 8)     default 0.00000000            not null,
    created_at   timestamp          default '2014-01-01 06:31:01' not null,
    updated_at   timestamp          default CURRENT_TIMESTAMP     not null on update CURRENT_TIMESTAMP,
    flag         tinyint(1)         default 1                     not null,
    wikiDataId   varchar(255)       default ''                    null comment 'Rapid API GeoDB Cities'
);

-- 插入一些基础测试数据（可选）
INSERT IGNORE INTO cities_test (name, state_id, state_code, country_id, country_code, latitude, longitude, flag, wikiDataId) VALUES
('Beijing', 1, 'BJ', 1, 'CN', 39.90420000, 116.40740000, 1, 'Q956'),
('Shanghai', 2, 'SH', 1, 'CN', 31.23040000, 121.47370000, 1, 'Q8686'),
('Tokyo', 4, 'TK', 2, 'JP', 35.67620000, 139.65030000, 1, 'Q1490'),
('Osaka', 5, 'OS', 2, 'JP', 34.69370000, 135.50230000, 0, 'Q35765');