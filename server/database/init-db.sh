#!/bin/bash

# Global constants
SCRIPT_PREFIX="./scripts/"

function create_db() {
	echo "Create new database \"$STM_DB_DATABASE\""

	createdb -h $STM_DB_HOST -U $STM_DB_USERNAME $STM_DB_DATABASE

	if [ $? -ne 0 ]
	then
		echo
		echo "Error during database creation."
		echo "Abort."
		exit 1
	fi

	echo "Ok"
}

function execute() {
		echo "=============================="
		echo "Execute file: $1"

		# Check what script-type we have (actually what file extension the script has) and execute the script accordingly
		if [[ "$1" == *".sql" ]]
		then
			psql -q -v ON_ERROR_STOP=1 -h $STM_DB_HOST -U $STM_DB_USERNAME -f $1 $STM_DB_DATABASE
			OK=$?
		elif [[ "$1" == *".sh" ]]
		then
			$1
			OK=$?
		fi

	# Check return value
	if [ $OK -ne 0 ]
	then
		echo "Error during script $1"
		echo "Abort."
		exit 1
		fi

		echo "Ok"
}

echo "Initialize database $STM_DB_DATABASE"

# First check if database exists
psql -h $STM_DB_HOST -U $STM_DB_USERNAME -lqt | cut -d \| -f 1 | grep -qw "$STM_DB_DATABASE"
DATABASE_EXISTS=$?

# Loop over all relevant files
FILES=$(ls $SCRIPT_PREFIX | tr " " "\n" | grep --color=never -P "^[[:digit:]]{3}" | tr "\n" " ")

for FILE in $FILES
do
	VERSION=$(echo $FILE | grep --color=never -Po "^[[:digit:]]{3}")

	if [ $DATABASE_EXISTS -ne 0 ] && [ "$VERSION" == "000" ]
	then # Database does not exist and we're looking at the init script => so execute initial script
		echo "Database $STM_DB_DATABASE does not exist. I'll create it."
		create_db
		execute $SCRIPT_PREFIX$FILE
	else # Database does exist and we're not looking at the init script => check if this script needs to be executed
		VERSION_ALREADY_APPLIED=$(psql -h $STM_DB_HOST -U $STM_DB_USERNAME $STM_DB_DATABASE -tc "SELECT * FROM db_versions WHERE version='$VERSION';" | sed '/^$/d' | wc -l)
		if [ $VERSION_ALREADY_APPLIED -eq 0 ]
		then
			execute $SCRIPT_PREFIX$FILE
		else
			echo "=============================="
			echo "Skip $VERSION: File $FILE already applied"
		fi
	fi
done

echo "=============================="
echo
echo "Done."