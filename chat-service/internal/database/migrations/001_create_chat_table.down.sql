DROP INDEX IF EXISTS idx_messages_room_id;
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_room_participants_room_id;

DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS room_participants;