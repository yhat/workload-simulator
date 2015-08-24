DELIMITER //

-- Schema & account creation
CREATE SCHEMA scienceops CHARACTER SET 'utf8'//

-- Create a scienceops user if none exists
GRANT SELECT, INSERT, EXECUTE, UPDATE, DELETE ON
    scienceops.* TO 'scienceops'@'%' IDENTIFIED BY 'scienceops'//

USE scienceops//

-- Installation metadata
CREATE TABLE ScienceOpsInstall (
    install_id   INTEGER PRIMARY KEY AUTO_INCREMENT,
    version      CHAR(10) NOT NULL UNIQUE,
    install_date DATETIME NOT NULL
) ENGINE=InnoDB//

INSERT INTO ScienceOpsInstall (version, install_date) VALUES ('2.0.0', NOW())//

-- User
CREATE TABLE User (
    user_id  INTEGER PRIMARY KEY AUTO_INCREMENT,
    username CHAR(250) NOT NULL UNIQUE,
    password CHAR(250) NOT NULL,
    email    CHAR(250) NOT NULL UNIQUE,
    admin    BOOLEAN NOT NULL DEFAULT FALSE,
    active   BOOLEAN NOT NULL DEFAULT TRUE
) ENGINE=InnoDB//

-- API Keys (add foreign key to models)
CREATE TABLE ApiKey (
    user_id          INTEGER NOT NULL,
    apikey 	         CHAR(250) NOT NULL UNIQUE,
    read_only_apikey CHAR(250) NOT NULL UNIQUE DEFAULT "",
    FOREIGN KEY (user_id) REFERENCES User(user_id) ON DELETE CASCADE
) ENGINE=InnoDB//

-- Model
CREATE TABLE Model (
    model_id       INTEGER PRIMARY KEY AUTO_INCREMENT,
    modelname      CHAR(250) NOT NULL,
    user_id        INTEGER NOT NULL,
    active_version INTEGER NOT NULL DEFAULT 0, -- this will always be more than zero
    deployment_id  INTEGER NOT NULL DEFAULT 0, -- internal deployment_id to fetch logs with
    status         CHAR(250) NOT NULL DEFAULT 'deployed',
    example_input  TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES User(user_id) ON DELETE CASCADE,
    CONSTRAINT model_uniq UNIQUE(modelname, user_id)
) ENGINE=InnoDB//

-- latest version
CREATE TABLE ModelVersion (
    id         INTEGER PRIMARY KEY AUTO_INCREMENT,
    model_id   INTEGER NOT NULL,
    version    INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    code       TEXT NOT NULL,
    lang  	   CHAR(250) NOT NULL,
    bundle     TEXT NOT NULL, -- bundle filename relative to the bundle directory
    FOREIGN KEY (model_id) REFERENCES Model(model_id) ON DELETE CASCADE,
    CONSTRAINT model_version_uniq UNIQUE(model_id, version)
) ENGINE=InnoDB//

-- Container represents a container running on a worker
-- ids are first reserved then the version is set later. Between those two
-- actions version will be NULL.
CREATE TABLE Container (
    id        INTEGER PRIMARY KEY AUTO_INCREMENT,
    deploy_id INTEGER
) ENGINE=InnoDB//

-- Stores data for model's shared users
CREATE TABLE ModelSharedUser (
    model_id       INTEGER NOT NULL,
    shared_user_id INTEGER NOT NULL,
    FOREIGN KEY (shared_user_id) REFERENCES User(user_id),
    FOREIGN KEY (model_id) REFERENCES Model(model_id) ON DELETE CASCADE,
    CONSTRAINT shared_user_uniq UNIQUE(model_id, shared_user_id)
) ENGINE=InnoDB//

-- A package is a package associated with a model version.
CREATE TABLE Package (
    version_id INTEGER,
    name       CHAR(250) NOT NULL,
    version    CHAR(250) NOT NULL,
    lang       CHAR(250) NOT NULL, -- r, python2, or apt-get
    FOREIGN KEY (version_id) REFERENCES ModelVersion(id) ON DELETE CASCADE
) ENGINE=InnoDB//

-- Server data
CREATE TABLE MPS (
    id         INTEGER PRIMARY KEY AUTO_INCREMENT,
    hostname   CHAR(250) NOT NULL UNIQUE,
    CONSTRAINT mps_hostname_uniq UNIQUE(hostname)
) ENGINE=InnoDB//

CREATE TABLE DockerBaseImages (
    id     INTEGER NOT NULL,
    image_name         TEXT NOT NULL,
    lang       CHAR(250) NOT NULL, 
    CONSTRAINT lang_uniq UNIQUE(lang)
) ENGINE=InnoDB//
