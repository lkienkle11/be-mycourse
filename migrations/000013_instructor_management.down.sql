DELETE FROM role_permissions
WHERE permission_id BETWEEN 'P41' AND 'P58';

DELETE FROM permissions WHERE permission_id BETWEEN 'P41' AND 'P58';

DROP TABLE IF EXISTS instructor_ticket_messages;
DROP TABLE IF EXISTS instructor_tickets;
DROP TABLE IF EXISTS instructor_expertise_skills;
DROP TABLE IF EXISTS instructor_expertise_topics;
DROP TABLE IF EXISTS instructor_profiles;
DROP TABLE IF EXISTS instructor_applications;

ALTER TABLE users DROP COLUMN IF EXISTS phone;
