--SELECT array(
SELECT DISTINCT o.file_id
FROM (
       SELECT DISTINCT
         s.file_id,
         sk.id
       FROM
         keywords k
         JOIN sticker_keywords sk ON sk.keyword_id = k.id
         JOIN stickers s ON sk.sticker_id = s.id
       WHERE sk.group_id = 1
             AND k.keyword ILIKE '%'
       ORDER BY sk.id DESC, s.file_id) o
LIMIT 51
OFFSET 50;

--SELECT array(
SELECT DISTINCT ON (sk.id) s.file_id
FROM
  keywords k
  JOIN sticker_keywords sk ON sk.keyword_id = k.id
  JOIN stickers s ON sk.sticker_id = s.id
WHERE sk.group_id = 1
      AND k.keyword ILIKE '%'
ORDER BY sk.id DESC;
--    LIMIT 51
--OFFSET 50);


--SELECT array(
select f.file_id
from (
       SELECT
         s.file_id,
         MAX(sk.id)
       FROM
         keywords k
         JOIN sticker_keywords sk ON sk.keyword_id = k.id
         JOIN stickers s ON sk.sticker_id = s.id
       WHERE sk.group_id = 1
             AND k.keyword ILIKE '%'
       GROUP BY s.file_id) as f;
--    LIMIT 51
--    OFFSET 50);


SELECT array(
    select f.file_id
    from (
           SELECT
             s.file_id,
             MAX(sk.id)
           FROM
             keywords k
             JOIN sticker_keywords sk ON sk.keyword_id = k.id
             JOIN stickers s ON sk.sticker_id = s.id
           WHERE sk.group_id = 1
                 AND k.keyword ILIKE '%'
           GROUP BY s.file_id) as f
    LIMIT 51
);

--SELECT array(
select f.file_id
from (
       SELECT
         s.file_id,
         MAX(sk.id)
       FROM
         keywords k
         JOIN sticker_keywords sk ON sk.keyword_id = k.id
         JOIN stickers s ON sk.sticker_id = s.id
       WHERE sk.group_id = 1
             AND k.keyword ILIKE '%'
       GROUP BY s.file_id) as f
LIMIT 51;
--);


SELECT
  i1.file_id                   as file_id,
  MAX(i1.id),
  row_number()
  OVER (
    ORDER BY MAX(i1.id) DESC ) AS rn
FROM
  (
    SELECT
      s.file_id,
      sk.id
    FROM keywords k
      JOIN sticker_keywords sk ON sk.keyword_id = k.id
      JOIN stickers s ON sk.sticker_id = s.id
    WHERE sk.group_id = 1
          AND k.keyword ILIKE 'k%'
  ) i1
INTERSECT
SELECT
  i2.file_id                   as file_id,
  MAX(i2.id),
  row_number()
  OVER (
    ORDER BY MAX(i2.id) DESC ) AS rn
FROM
  (
    SELECT
      s.file_id,
      sk.id
    FROM keywords k
      JOIN sticker_keywords sk ON sk.keyword_id = k.id
      JOIN stickers s ON sk.sticker_id = s.id
    WHERE sk.group_id = 1
          AND k.keyword ILIKE 'k%'
  ) i2

GROUP BY file_id
ORDER by MAX(i1.id) DESC
LIMIT 50
OFFSET 50