/* Script to bootstrap  posgresql DB running in container */
CREATE DATABASE defaultdb;
\c defaultdb;
GRANT ALL PRIVILEGES ON DATABASE defaultdb TO uc_root_user;
CREATE TABLE tmp (id UUID);
DROP TABLE tmp;
