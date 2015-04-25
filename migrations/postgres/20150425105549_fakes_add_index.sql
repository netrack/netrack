
-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE INDEX idxfakeid ON fakes USING btree ((fake->>'id'));

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP INDEX IF EXISTS idfakeid;
