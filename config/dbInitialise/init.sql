CREATE TABLE users (
    user_id bigint,
    username text,
    invited_by bigint,
    chat_id bigint,
    points integer,
    openart boolean,
    planets boolean
);