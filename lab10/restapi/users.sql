CREATE SEQUENCE IF NOT EXISTS users_id_seq;

CREATE TABLE IF NOT EXISTS users (
  id INTEGER NOT NULL DEFAULT nextval('users_id_seq') PRIMARY KEY,
  name TEXT NOT NULL,
  age INTEGER NOT NULL,
  comment TEXT,
  email TEXT NOT NULL,
  phone TEXT,
  role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
  password_hash TEXT NOT NULL DEFAULT ''
);

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';

ALTER TABLE users
  ALTER COLUMN id SET DEFAULT nextval('users_id_seq');

ALTER SEQUENCE users_id_seq OWNED BY users.id;
