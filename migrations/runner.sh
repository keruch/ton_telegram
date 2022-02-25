#!/bin/bash

set -e
export CMD=$1
readonly CONF=migrations/dbconfig.yml

if [ "$CMD" == "up" ]
    then
    # --dryrun - don't apply migrations, just print them
    echo "Applying migrations"
    sql-migrate up --config=$CONF --env=$ENV --dryrun
    sql-migrate up --config=$CONF --env=$ENV
    echo "Done"
    echo "Migrations status"
    sql-migrate status --config=$CONF --env=$ENV
    echo "Done"

elif [ "$CMD" == "down" ]
    then
    # --dryrun - don't apply migrations, just print them
    echo "Down migrations"
    sql-migrate down --limit=1 --config=$CONF --env=$ENV --dryrun
    sql-migrate down --limit=1 --config=$CONF --env=$ENV
    echo "Done"
    echo "Migrations status"
    sql-migrate status --config=$CONF --env=$ENV
    echo "Done"

elif [ "$CMD" == "status" ]
    then
    echo "Migrations status"
    sql-migrate status --config=$CONF --env=$ENV
    echo "Done"

elif [ "$CMD" == "up-down-up" ]
    then
    echo "Applying migrations"
    sql-migrate up --config=$CONF --env=$ENV --dryrun
    MIGRATIONS_COUNT=$(sql-migrate up --config=$CONF --env=$ENV | sed 's/.*Applied \(.*\) migration.*/\1/')
    re='^[0-9]+$'
    if ! [[ $MIGRATIONS_COUNT =~ $re ]] 
    then
        echo "Something wrong with migration. Error: $MIGRATIONS_COUNT" >&2
        exit 1
    fi
    echo "$MIGRATIONS_COUNT migrations was applied."
    if [ $MIGRATIONS_COUNT != "0" ]
    then 
        echo "Rolling them back"
        sql-migrate down --config=$CONF --env=$ENV --limit=$MIGRATIONS_COUNT --dryrun
        sql-migrate down --config=$CONF --env=$ENV --limit=$MIGRATIONS_COUNT
        echo "And applying them again"
        sql-migrate up --config=$CONF --env=$ENV --dryrun
        sql-migrate up --config=$CONF --env=$ENV
        echo "Done"
    else
        echo "Looks like everything is up to date. Nothing to do here!"
    fi
else
    echo "Incorrect command. Use up or up-down-up"
    exit 1 
fi