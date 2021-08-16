#!/bin/sh

sqlite3 sample_db 'CREATE TABLE person(id INTEGER PRIMARY KEY, name TEXT);'
sqlite3 sample_db 'INSERT INTO person(id,name) VALUES(1,"PersonA");'
sqlite3 sample_db 'INSERT INTO person(id,name) values(2,"PersonB");'
sqlite3 sample_db 'INSERT INTO person(id,name) values(3,"PersonC");'
sqlite3 sample_db 'INSERT INTO person(id,name) values(4,"PersonD");'
sqlite3 sample_db 'INSERT INTO person(id,name) values(5,"PersonE");'
