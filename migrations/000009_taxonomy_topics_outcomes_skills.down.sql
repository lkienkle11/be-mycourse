DELETE FROM role_permissions
WHERE permission_id IN ('P30', 'P31', 'P32', 'P33', 'P34', 'P35', 'P36', 'P37');

DELETE FROM permissions
WHERE permission_id IN ('P30', 'P31', 'P32', 'P33', 'P34', 'P35', 'P36', 'P37');

DROP TABLE IF EXISTS course_skills;

DROP TABLE IF EXISTS course_outcomes;

UPDATE permissions SET permission_name = 'category:read', updated_at = NOW() WHERE permission_id = 'P18';

UPDATE permissions SET permission_name = 'category:create', updated_at = NOW() WHERE permission_id = 'P19';

UPDATE permissions SET permission_name = 'category:update', updated_at = NOW() WHERE permission_id = 'P20';

UPDATE permissions SET permission_name = 'category:delete', updated_at = NOW() WHERE permission_id = 'P21';

ALTER TABLE course_topics DROP COLUMN IF EXISTS child_topics;

ALTER INDEX idx_course_topics_image_file_id RENAME TO idx_categories_image_file_id;

ALTER INDEX idx_course_topics_created_by RENAME TO idx_categories_created_by;

ALTER TABLE course_topics RENAME TO categories;
