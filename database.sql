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
    -- system_role dictates global permissions (can they create teams?)
    system_role ENUM('manager', 'member') NOT NULL DEFAULT 'member',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
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
-- 4. INDEXES (For Query Optimization)
-- ==========================================
-- Speeds up login checks
CREATE INDEX idx_users_email ON users(email);
-- Speeds up "Find all teams for User X" queries
CREATE INDEX idx_team_members_user ON team_members(user_id);