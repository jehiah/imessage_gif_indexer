#!/bin/bash

TARGET=...
for FILE in $(find ~/Library/Messages/Attachments -name "output*.GIF"); do
	CREATED_TS=$(stat -f %c -t %s $FILE)
	MODIFIED_TS=$(stat -f %m -t %s $FILE)
    TIME=$CREATED_TS
    if [ $MODIFIED_TS -lt $CREATED_TS ]; then
        TIME=$MODIFIED_TS
    fi
    # SUFFIX=$( strings < /dev/urandom | tr -dc A-Za-z0-9 2>/dev/null| head -c6)
    SUFFIX=$(dd if=/dev/urandom bs=1 count=32 2>/dev/null | LC_CTYPE=C tr -dc A-Za-z0-9 | head -c6)
	NEW_FILE="$(date -r $TIME +%Y%m%d_%H%M%S)_${SUFFIX}.gif"
	echo "$FILE -> $NEW_FILE"
    cp $FILE $TARGET/${NEW_FILE}
done

