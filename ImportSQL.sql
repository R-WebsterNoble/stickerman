-- Data from https://e621.net/db_export/

TRUNCATE TABLE public.tag_implications

SELECT id, antecedent_name, consequent_name, creator_id, creator_ip_addr, forum_topic_id, status, created_at, updated_at, approver_id, forum_post_id, descendant_names, reason
	FROM public.tag_implications;

ALTER TABLE public.tag_implications
ALTER COLUMN creator_id SET DEFAULT  1;

ALTER TABLE public.tag_implications
ALTER COLUMN creator_ip_addr SET DEFAULT  '127.0.0.1';

COPY public.tag_implications ( id, antecedent_name, consequent_name, created_at, status )
FROM '/tmp/tag_implications-2024-02-07.csv'
DELIMITER ','
CSV HEADER;

UPDATE public.tag_implications SET approver_id = 1;
UPDATE public.tag_implications SET status = 'active' WHERE status = 'pending'; 

----------------------------------------

TRUNCATE TABLE public.tag_aliases

SELECT id, antecedent_name, consequent_name, creator_id, creator_ip_addr, forum_topic_id, status, created_at, updated_at, post_count, approver_id, forum_post_id, reason
	FROM public.tag_aliases;


ALTER TABLE public.tag_aliases
ALTER COLUMN creator_id SET DEFAULT  1;

ALTER TABLE public.tag_aliases
ALTER COLUMN creator_ip_addr SET DEFAULT  '127.0.0.1';

COPY public.tag_aliases ( id, antecedent_name, consequent_name, created_at, status )
FROM '/tmp/tag_aliases-2024-02-07.csv'
DELIMITER ','
CSV HEADER;

UPDATE public.tag_aliases SET approver_id = 1;
UPDATE public.tag_aliases SET status = 'active' WHERE status = 'pending'; 