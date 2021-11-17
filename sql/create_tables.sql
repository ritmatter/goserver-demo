CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS channels (
  channel_id uuid DEFAULT uuid_generate_v4 (),
  name varchar(250) NOT NULL,
  PRIMARY KEY (channel_id)
);

CREATE TABLE IF NOT EXISTS messages (
  message_id uuid DEFAULT uuid_generate_v4 (),
  channel_id uuid,
  text varchar(250) NOT NULL,
  PRIMARY KEY (message_id),
  CONSTRAINT fk_channel
    FOREIGN KEY(channel_id)
  REFERENCES channels(channel_id)
);
