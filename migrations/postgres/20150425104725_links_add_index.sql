
-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE INDEX idxlinkid ON links USING btree ((link->>'id'));

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP INDEX IF EXISTS idlinkid;
