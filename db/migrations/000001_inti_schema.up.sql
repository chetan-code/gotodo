CREATE TABLE IF NOT EXISTS todos (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    title TEXT NOT NULL,
    is_done BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Optimization: Indexing the column used in your WHERE clauses
CREATE INDEX IF NOT EXISTS idx_todos_email ON todos(email);