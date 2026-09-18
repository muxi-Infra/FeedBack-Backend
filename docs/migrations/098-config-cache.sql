-- MySQL 8.0 equivalent of the additive AutoMigrate upgrade.
-- Apply the ALTER once to an existing V3 schema, before deploying new writers.
-- AutoMigrate checks column existence and is repeatable.
ALTER TABLE feedback_projects ADD COLUMN config_version BIGINT UNSIGNED NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS feedback_project_config_audits (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  change_id VARCHAR(36) NOT NULL,
  project_id VARCHAR(64) NOT NULL,
  admin_id BIGINT UNSIGNED NOT NULL,
  request_id VARCHAR(64) NOT NULL,
  kind VARCHAR(32) NOT NULL,
  previous_version BIGINT UNSIGNED NOT NULL,
  version BIGINT UNSIGNED NOT NULL,
  fields VARCHAR(255) NOT NULL,
  result VARCHAR(16) NOT NULL,
  created_at DATETIME(3) NULL,
  UNIQUE KEY idx_feedback_project_config_audits_change_id (change_id),
  KEY idx_feedback_project_config_audits_project_id (project_id),
  KEY idx_feedback_project_config_audits_request_id (request_id)
);

CREATE TABLE IF NOT EXISTS feedback_project_config_outbox (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  change_id VARCHAR(36) NOT NULL,
  project_id VARCHAR(64) NOT NULL,
  version BIGINT UNSIGNED NOT NULL,
  kind VARCHAR(32) NOT NULL,
  created_at DATETIME(3) NULL,
  available_at DATETIME(3) NOT NULL,
  published_at DATETIME(3) NULL,
  lease_owner VARCHAR(128) NOT NULL DEFAULT '',
  attempts BIGINT NOT NULL DEFAULT 0,
  last_error VARCHAR(32) NOT NULL DEFAULT '',
  message_id VARCHAR(64) NOT NULL DEFAULT '',
  UNIQUE KEY idx_feedback_project_config_outbox_change_id (change_id),
  KEY idx_v3_outbox_due (published_at, available_at)
);
