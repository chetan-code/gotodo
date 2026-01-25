-- Remove the index and column from todos
DROP INDEX IF EXISTS idx_todos_worker;
ALTER TABLE todos DROP COLUMN worker_email;

-- Remove the relationships table
DROP INDEX IF EXISTS idx_relationships_worker;
DROP INDEX IF EXISTS idx_relationships_manager;
DROP TABLE IF EXISTS relationships;