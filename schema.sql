-- noinspection SpellCheckingInspectionForFile

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

create table stickers
(
  id      bigserial not null
    constraint unique_stickers_id
    primary key,
  file_id text      not null
    constraint unique_stickers_file_id
    unique
);

create table keywords
(
  id      bigserial not null
    constraint unique_keywords_id
    primary key,
  keyword text      not null
    constraint unique_keywords_keyword
    unique
);

create table groups
(
  id   bigserial                       not null
    constraint unique_groups_id
    unique,
  uuid uuid default uuid_generate_v4() not null
    constraint unique_groups_uuid
    unique
);

create table sticker_keywords
(
  id         bigserial not null
    constraint unique_sticker_keywords_id
    primary key,
  sticker_id bigint    not null
    constraint lnk_stickers_sticker_keywords
    references stickers
    on update cascade on delete cascade,
  keyword_id bigint    not null
    constraint lnk_keywords_sticker_keywords
    references keywords
    on update cascade on delete cascade,
  group_id   bigint    not null
    constraint lnk_groups_sticker_keywords
    references groups (id)
    on update cascade on delete cascade,
  constraint stickerkeywords_stickerid_keywordid_groupid
  unique (sticker_id, keyword_id, group_id)
);

create index index_sticker_id
  on sticker_keywords (sticker_id);

create index index_keyword_id
  on sticker_keywords (keyword_id);

create table sessions
(
  id       bigserial                                      not null
    constraint unique_sessions_id
    primary key,
  chat_id  bigint                                         not null
    constraint unique_sessions_chat_id
    unique,
  file_id  text,
  mode     varchar(20) default 'add' :: character varying not null,
  group_id bigint                                         not null
    constraint lnk_groups_sessions
    references groups (id)
    on update cascade on delete cascade
);

create index index_file_id
  on sessions (file_id);

create index index_mode
  on sessions (mode);

create function fn_please_dont_use_me(VARIADIC input_keywords text [])
  returns TABLE(id bigint, file_id text)
language plpgsql
as $$
DECLARE
  input_keyword_id     BIGINT;
  matching_sticker_ids BIGINT [];
BEGIN
  SELECT INTO matching_sticker_ids ARRAY(SELECT s.id
                                         FROM stickers s);

  RAISE NOTICE '1 %', matching_sticker_ids;
  RAISE NOTICE 'k %', input_keywords;
  FOR input_keyword_id IN SELECT k.id
                          from keywords k
                          where k.keyword ILIKE ANY (input_keywords)
  LOOP
    SELECT INTO matching_sticker_ids ARRAY(
        SELECT DISTINCT s.id
        FROM stickers s
          JOIN sticker_keywords sk ON sk.sticker_id = s.id AND sk.keyword_id = input_keyword_id
        WHERE
          s.id = ANY (matching_sticker_ids)
    );
    RAISE NOTICE '2 %', input_keyword_id;
    RAISE NOTICE '3 %', matching_sticker_ids;
  END LOOP;
  RETURN QUERY SELECT *
               FROM stickers s
               WHERE s.id = ANY (matching_sticker_ids);
END;
$$;
