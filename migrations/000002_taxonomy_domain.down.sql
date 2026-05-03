DELETE FROM permissions
WHERE permission_id IN ('P14', 'P15', 'P16', 'P17', 'P18', 'P19', 'P20', 'P21', 'P22', 'P23', 'P24', 'P25');

DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS course_levels;

DROP TYPE IF EXISTS taxonomy_status;
