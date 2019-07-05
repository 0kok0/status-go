CREATE TABLE IF NOT EXISTS browsers (
id TEXT PRIMARY KEY,
name TEXT NOT NULL,
timestamp USGIGNED BIGINT,
dapp BOOLEAN DEFAULT false,
historyIndex UNSIGNED INT
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS browsers_history (
browser_id TEXT NOT NULL,
history TEXT,
FOREIGN KEY(browser_id) REFERENCES browsers(id) ON DELETE CASCADE
)