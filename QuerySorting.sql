SELECT ARRAY(
    SELECT f.fid
    FROM (
           SELECT
             t1.fid,
             MAX(t1.skid),
             row_number()
             OVER (
               ORDER BY MAX(t1.skid) DESC ) AS rn
           FROM
             (
               SELECT
                 s.id      AS sid,
                 s.file_id AS fid,
                 sk.id     AS skid
               FROM keywords k
                 JOIN sticker_keywords sk ON sk.keyword_id = k.id
                 JOIN stickers s ON sk.sticker_id = s.id
               WHERE sk.group_id = 1 --$1
                     AND k.keyword ILIKE 'alpha' --$2
             ) t1
             JOIN
             (
               SELECT
                 s.id as sid2,
                 s.file_id,
                 sk.id
               FROM keywords k
                 JOIN sticker_keywords sk ON sk.keyword_id = k.id
                 JOIN stickers s ON sk.sticker_id = s.id
               WHERE sk.group_id = 1
                     AND k.keyword ILIKE 'bravo%' --$3
             ) t2 ON t1.sid = t2.sid2

           GROUP BY t1.fid
           ORDER by MAX(t1.skid) DESC
           LIMIT 50
           OFFSET 0
         ) AS f
    ORDER BY f.rn
);