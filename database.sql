-- Create the database (if it doesn't already exist)
CREATE DATABASE IF NOT EXISTS microservices_capstone;
USE microservices_capstone;

-- ==========================================
-- 1. USERS TABLE
-- ==========================================
CREATE TABLE IF NOT EXISTS users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    system_role ENUM('manager', 'member') NOT NULL DEFAULT 'member',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

 INSERT INTO users (username, email, password_hash, system_role)
 SELECT 'Admin', 'admin@example.com', '$2a$10$Ylqycc.m0K2oJkvdomU5f.6Yb8.sco0oQWF8r36J76ideDPS.bCGO', 'manager'
 WHERE NOT EXISTS (
     SELECT 1
     FROM users
     WHERE email = 'admin@example.com'
 );

-- ==========================================
-- 2. TEAMS TABLE
-- ==========================================
CREATE TABLE IF NOT EXISTS teams (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- ==========================================
-- 3. TEAM MEMBERS (JUNCTION TABLE)
-- ==========================================
-- This table manages the Many-to-Many relationship and team-specific roles.
CREATE TABLE IF NOT EXISTS team_members (
    team_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    -- team_role dictates local permissions inside a specific team
    team_role ENUM('main_manager', 'manager', 'member') NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Composite primary key ensures a user can only be added to a team once
    PRIMARY KEY (team_id, user_id),

    -- Foreign keys with CASCADE ensure that deleting a user or team
    -- automatically removes their membership records, preventing orphaned data.
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ==========================================
-- 4. FOLDERS TABLE
-- ==========================================
CREATE TABLE IF NOT EXISTS folders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    owner_id BIGINT NOT NULL, -- FK to users
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ==========================================
-- 5. NOTES TABLE
-- ==========================================
CREATE TABLE IF NOT EXISTS notes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    folder_id BIGINT NOT NULL, -- FK to folders
    owner_id BIGINT NOT NULL,  -- FK to users
    title VARCHAR(255) NOT NULL,
    content TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ==========================================
-- 6. FOLDER_SHARES TABLE (Junction table for sharing folders with users)
-- ==========================================
CREATE TABLE IF NOT EXISTS folder_shares (
    folder_id BIGINT NOT NULL,
    shared_with_user_id BIGINT NOT NULL,
    permission_level ENUM('read', 'write') NOT NULL,
    PRIMARY KEY (folder_id, shared_with_user_id),
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE,
    FOREIGN KEY (shared_with_user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ==========================================
-- 7. NOTE_SHARES TABLE (Junction table for sharing notes with users)
-- ==========================================
CREATE TABLE IF NOT EXISTS note_shares (
    note_id BIGINT NOT NULL,
    shared_with_user_id BIGINT NOT NULL,
    permission_level ENUM('read', 'write') NOT NULL,
    PRIMARY KEY (note_id, shared_with_user_id),
    FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE,
    FOREIGN KEY (shared_with_user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ==========================================
-- 8. AUDIT_LOGS TABLE (For tracking user actions)
-- ==========================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NULL,
    action VARCHAR(255) NOT NULL,
    entity_type VARCHAR(100) NULL,
    entity_id BIGINT NULL,
    details JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- ==========================================
-- 9. INDEXES (For Query Optimization)
-- ==========================================
-- Speeds up login checks
CREATE INDEX idx_users_email ON users(email);
-- Speeds up "Find all teams for User X" queries
CREATE INDEX idx_team_members_user ON team_members(user_id);