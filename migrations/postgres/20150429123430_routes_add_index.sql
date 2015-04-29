
-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE INDEX idxrouteid ON routes USING btree ((route->>'id'));

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP INDEX IF EXISTS idrouteid;
