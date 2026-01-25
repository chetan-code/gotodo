-- Relationship table 
-- handlers the request (the manager send req to worker)
-- when accept he can assign task to worker
CREATE TABLE IF NOT EXISTS relationships (
        id SERIAL PRIMARY KEY,
        manager_email  TEXT NOT NULL,
        worker_email TEXT NOT NULL,
        status TEXT NOT NULL DEFAULT 'pending',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

        -- ENSURE WE CAN INVITE TWICE - UNIQUE ONE TO ONE RELATIONSHIP
        UNIQUE(manager_email, worker_email)
);

-- indices to spped loading "My Invites"
CREATE INDEX IF NOT EXISTS idx_relationships_manager ON relationships(manager_email);
CREATE INDEX IF NOT EXISTS idx_relationships_worker ON relationships(worker_email);


-- we need to assign task to workers
ALTER TABLE todos ADD COLUMN worker_email TEXT;

-- update todos to assign worker email to self
UPDATE todos SET worker_email = email;

-- we can now set it to not null
ALTER TABLE todos ALTER COLUMN worker_email SET NOT NULL;

-- tasks assigned to me
CREATE INDEX IF NOT EXISTS idx_todos_worker ON todos(worker_email);