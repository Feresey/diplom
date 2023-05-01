-- foreign keys
SELECT
	constraint_schema,
	constraint_name,
	unique_constraint_schema,
	unique_constraint_name,
	match_option,
	update_rule,
	delete_rule
FROM
	information_schema.referential_constraints
WHERE
	constraint_schema || '.' || constraint_name = ANY($1)
