-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS user_data (user_id bigint, invited_by bigint, points integer, openart boolean, additional boolean);

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE user_data;