-- 1. Create a dedicated database
CREATE DATABASE minurl;

-- 2. Create the Application User
-- CREATE USER web WITH ENCRYPTED PASSWORD 'qwe';

-- 3. Revoke default public permissions (Safety first)
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT CONNECT ON DATABASE minurl TO web;
GRANT USAGE ON SCHEMA public TO web;
