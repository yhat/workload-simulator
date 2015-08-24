DELIMITER //
USE scienceops//
INSERT INTO MpsServer(hostname) VALUES('123.0.0.0')//
INSERT INTO User(username, password, email) VALUES ('test1', 'foo', 'ec@yhathq.com1')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('1', 'abc1231', 'def12341')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld11', 1, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld12', 1, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld13', 1, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld14', 1, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld15', 1, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
INSERT INTO User(username, password, email) VALUES ('test2', 'foo', 'ec@yhathq.com2')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('2', 'abc1232', 'def12342')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld21', 2, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld22', 2, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld23', 2, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld24', 2, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld25', 2, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
INSERT INTO User(username, password, email) VALUES ('test3', 'foo', 'ec@yhathq.com3')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('3', 'abc1233', 'def12343')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld31', 3, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld32', 3, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld33', 3, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld34', 3, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld35', 3, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
INSERT INTO User(username, password, email) VALUES ('test4', 'foo', 'ec@yhathq.com4')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('4', 'abc1234', 'def12344')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld41', 4, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld42', 4, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld43', 4, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld44', 4, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld45', 4, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
INSERT INTO User(username, password, email) VALUES ('test5', 'foo', 'ec@yhathq.com5')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('5', 'abc1235', 'def12345')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld51', 5, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld52', 5, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld53', 5, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld54', 5, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld55', 5, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
INSERT INTO User(username, password, email) VALUES ('test6', 'foo', 'ec@yhathq.com6')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('6', 'abc1236', 'def12346')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld61', 6, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld62', 6, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld63', 6, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld64', 6, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld65', 6, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
INSERT INTO User(username, password, email) VALUES ('test7', 'foo', 'ec@yhathq.com7')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('7', 'abc1237', 'def12347')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld71', 7, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld72', 7, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld73', 7, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld74', 7, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld75', 7, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
INSERT INTO User(username, password, email) VALUES ('test8', 'foo', 'ec@yhathq.com8')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('8', 'abc1238', 'def12348')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld81', 8, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld82', 8, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld83', 8, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld84', 8, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld85', 8, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
INSERT INTO User(username, password, email) VALUES ('test9', 'foo', 'ec@yhathq.com9')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('9', 'abc1239', 'def12349')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld91', 9, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld92', 9, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld93', 9, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld94', 9, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld95', 9, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
INSERT INTO User(username, password, email) VALUES ('test10', 'foo', 'ec@yhathq.com10')//
INSERT INTO ApiKey(user_id, apikey, read_only_apikey) VALUES ('10', 'abc12310', 'def123410')//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld101', 10, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(1, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(1, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (1, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld102', 10, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(2, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(2, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (2, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld103', 10, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(3, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(3, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (3, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld104', 10, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(4, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(4, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (4, 1, 1)//
INSERT INTO Model(modelname, user_id, lang) VALUES ('HelloWorld105', 10, 'python')//
INSERT INTO ModelVersion(model_id, version, code) VALUES(5, 1, 'print Hello World')//
INSERT INTO ModelStatus(model_id, status) VALUES(5, 'online')//
INSERT INTO MpsModelInstance(model_id, version, mps_id) VALUES (5, 1, 1)//
