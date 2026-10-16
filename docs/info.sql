-- show table sizes in a human-readable format

SELECT 
     schemaname AS schema_name,
     relname AS table_name,
     pg_size_pretty(pg_table_size(relid)) AS table_size,
     pg_size_pretty(pg_indexes_size(relid)) AS index_size,
     pg_size_pretty(pg_total_relation_size(relid)) AS total_size
 FROM pg_catalog.pg_stat_user_tables
 ORDER BY pg_total_relation_size(relid) DESC;

 