create table user_age_verification
(
    user_id bigint
);

alter table user_age_verification
    owner to stickerman;

create unique index user_age_verification_chat_id_uindex
    on user_age_verification (user_id);

