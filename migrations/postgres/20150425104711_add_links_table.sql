
-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE links (link json);

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE links;
