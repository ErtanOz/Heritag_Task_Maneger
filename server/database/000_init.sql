BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS db_versions(
    version TEXT NOT NULL
);

-- Store version of the database scheme
INSERT INTO db_versions VALUES('000');

END TRANSACTION;