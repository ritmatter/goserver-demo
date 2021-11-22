CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
  user_id uuid DEFAULT uuid_generate_v4 (),
  name varchar(64) UNIQUE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id)
);

CREATE TABLE IF NOT EXISTS channels (
  channel_id uuid DEFAULT uuid_generate_v4 (),
  name varchar(64) UNIQUE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (channel_id)
);

CREATE TABLE IF NOT EXISTS messages (
  message_id uuid DEFAULT uuid_generate_v4 (),
  channel_id uuid,
  text varchar(250) NOT NULL,
  user_id uuid,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (message_id),
  CONSTRAINT fk_channel
    FOREIGN KEY(channel_id)
  REFERENCES channels(channel_id),
  CONSTRAINT fk_user
    FOREIGN KEY(user_id)
  REFERENCES users(user_id)
);
CREATE INDEX messages_created_at_index ON messages (created_at DESC);
