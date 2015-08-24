DELIMITER //

USE scienceops//

-- UserSelectName finds a user by name.
CREATE PROCEDURE UserSelectName(IN u_name CHAR(250)) BEGIN
    SELECT u.user_id, u.username, u.password, u.email, u.admin, u.active, a.apikey, a.read_only_apikey
    FROM   User u
    INNER JOIN ApiKey a ON
        u.user_id = a.user_id
    WHERE  u.username = u_name;
END//

-- UserSelectAll selects all users from the database
CREATE PROCEDURE UserSelectAll() BEGIN
    SELECT u.user_id, u.username, u.password, u.email, u.admin, u.active, a.apikey, a.read_only_apikey
    FROM   User u
    INNER JOIN ApiKey a ON
        u.user_id = a.user_id
    ORDER BY u.username;
END//

-- ModelSelectByUser selects all models for a given user user_id
-- and the count of ModelVersions for that Model
CREATE PROCEDURE UserModels(IN u_username CHAR(250)) BEGIN
    SELECT m.model_id, m.modelname, COUNT(m.model_id), MAX(v.created_at), m.status
	FROM Model m
    INNER JOIN ModelVersion v
	ON v.model_id = m.model_id
    INNER JOIN User u
    ON m.user_id = u.user_id
    WHERE u.username = u_username
	GROUP BY m.model_id;
END//

-- Returns the last updated ModelVersion for a given Model
CREATE PROCEDURE ModelsSharedWithUser(IN user_id INTEGER) BEGIN
    SELECT
        m.model_id
    FROM
        Model m
    INNER JOIN ModelSharedUser ms ON
        m.model_id = ms.model_id
        and ms.shared_user_id = user_id;
END//

-- Returns the users shared for a given model
CREATE PROCEDURE ModelSharedUsers(IN m_id INTEGER) BEGIN
    SELECT u.user_id, u.username, mu.shared_user_id is not NULL as is_shared
    FROM User u
    LEFT JOIN Model m on
        m.model_id = m_id
    LEFT JOIN ModelSharedUser mu ON
        mu.shared_user_id = u.user_id
        AND mu.model_id = m_id
    WHERE
        m.user_id != u.user_id;
END//

-- updates the apikey
CREATE PROCEDURE UpdateApikey(IN user_id INTEGER, IN new_apikey TEXT) BEGIN
    UPDATE
        ApiKey a
    SET
        apikey = new_apikey
    WHERE
        a.user_id = user_id;
END//

-- updates the read-only-apikey
CREATE PROCEDURE UpdateReadOnlyApikey(IN user_id INTEGER, IN new_read_only_apikey TEXT) BEGIN
    UPDATE
        ApiKey a
    SET
        read_only_apikey = new_read_only_apikey
    WHERE
        a.user_id = user_id;
END//
