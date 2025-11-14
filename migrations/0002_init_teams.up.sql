CREATE TABLE IF NOT EXISTS teams (
    team_name VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE users
    ADD CONSTRAINT fk_users_team
    FOREIGN KEY (team_name)
    REFERENCES teams(team_name)
    ON DELETE SET NULL;