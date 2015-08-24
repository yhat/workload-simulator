DELIMITER //

USE scienceops//

-- Create a new version of model.
CREATE PROCEDURE NewModelVersion(
    IN m_modelname VARCHAR(250),
    IN m_user_id   INTEGER,
    IN m_lang      VARCHAR(250),
    IN m_code      TEXT,
    IN m_bundle    TEXT
) BEGIN

    INSERT IGNORE INTO Model
        (modelname, user_id, example_input)
    VALUES
        (m_modelname, m_user_id, '{"input":"here"}');

    -- select model id
    SET @model_id = (
        SELECT m.model_id 
        FROM Model m
        WHERE m.user_id = m_user_id AND m.modelname = m_modelname
    );

    SET @model_version = (SELECT IFNULL(MAX(version), 0)
        AS maxversion
        FROM ModelVersion
        WHERE model_id = @model_id
    );

    INSERT INTO ModelVersion
        (model_id, code, version, bundle, lang, created_at)
    VALUES
        (@model_id, m_code, @model_version+1, m_bundle, m_lang, NOW());

    SET @model_version_id = LAST_INSERT_ID();

    UPDATE Model
    SET active_version = @model_version+1
    WHERE model_id = @model_id;

    SELECT mv.id, mv.version FROM ModelVersion mv where mv.id = @model_version_id;
END//

CREATE PROCEDURE GetModelVersion(IN
    u_username  CHAR(250),
    u_modelname CHAR(250),
    u_version   INTEGER
) BEGIN
    SELECT v.id, v.model_id, v.created_at, v.code, v.bundle, v.lang
    FROM ModelVersion v
    WHERE v.version = u_version AND v.model_id IN (
        SELECT m.model_id
        FROM Model m
        WHERE m.modelname = u_modelname AND m.user_id IN (
            SELECT u.user_id
            FROM User u
            WHERE u.username = u_username
        )
    );
END//

CREATE PROCEDURE NewDeployment(IN
    u_username  CHAR(250),
    u_modelname CHAR(250),
    u_version   INTEGER
) BEGIN

    SET @max = (SELECT IFNULL(MAX(deployment_id), 0) FROM Model);

    SET @m_model_id = (
        SELECT m.model_id
	    FROM Model m
	    INNER JOIN User u
	    ON m.user_id = u.user_id
	    WHERE m.modelname = u_modelname AND u.username = u_username
    );

    UPDATE Model
    SET deployment_id = @max + 1,
        active_version = u_version
    WHERE model_id = @m_model_id;

    SELECT deployment_id
    FROM Model
    WHERE model_id = @m_model_id;
END//

-- get model by id
CREATE PROCEDURE ActiveModelById(IN model_id INTEGER) BEGIN
    SELECT
        m.model_id
        , m.modelname
        , mv.version
        , mv.created_at
        , mv.code
        , mv.lang
    FROM Model m
    INNER JOIN ModelVersion mv ON
        m.model_id = mv.model_id
    WHERE
        m.model_id = model_id
        and mv.version = (
            SELECT version FROM ModelVersion order by created_at desc LIMIT 1
        )
    LIMIT 1;
END//

-- fetches versions for a given model_id
CREATE PROCEDURE VersionsByModel(IN model_id INTEGER) BEGIN
    SELECT
        m.model_id
        , m.modelname
        , mv.version
        , mv.created_at
        , mv.code
    FROM Model m
    INNER JOIN ModelVersion mv ON
        m.model_id = mv.model_id
    WHERE
        m.model_id = model_id
    ORDER by
        mv.created_at DESC;
END//

CREATE PROCEDURE AddModelSharedUser(IN m_id INTEGER, IN u_id INTEGER) BEGIN
    INSERT INTO ModelSharedUser(model_id, shared_user_id) values(m_id, u_id);
END//

CREATE PROCEDURE RemoveModelSharedUser(IN m_id INTEGER, IN u_id INTEGER) BEGIN
    DELETE FROM
        ModelSharedUser
    WHERE
        model_id = m_id AND shared_user_id = u_id;
END//

-- update a ModelStatus 
CREATE PROCEDURE SetModelStatus(IN m_id INTEGER, IN m_status CHAR(250)) BEGIN
    -- this is an upsert
    UPDATE Model
    SET status = m_status
    WHERE model_id = m_id;
END//
