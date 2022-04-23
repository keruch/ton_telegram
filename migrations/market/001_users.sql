-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE users
(
    id        bigint,
    username  text,
    nft_count int            DEFAULT 0,
    nft_id    int[] NOT NULL DEFAULT array []::int[],
    PRIMARY KEY (id)
);

-- +migrate StatementBegin
CREATE FUNCTION count_update() RETURNS trigger AS
$$
BEGIN
    NEW.nft_count = array_length(NEW.nft_id, 1);
    IF NEW.nft_count IS NULL THEN
        NEW.nft_count = 0;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql' SECURITY DEFINER;
-- +migrate StatementEnd

CREATE TRIGGER count_update
    BEFORE INSERT OR UPDATE
    ON users
    FOR EACH ROW
EXECUTE PROCEDURE count_update();
-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TRIGGER count_update ON users;
DROP FUNCTION count_update;
DROP TABLE users;