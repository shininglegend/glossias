#!/bin/bash

# deploy-annotator.sh
set -e  # Exit on any error

# Directory setup
ANNOTATOR_DIR="annotator"
STATIC_DIR="static/admin/annotator"
# CONFIG_FILE="src/config.tsx"

# Step 1: CD to annotator
cd $ANNOTATOR_DIR

# Step 2: Update config for production
# sed -i.bak 's|/\$|/|' $CONFIG_FILE

# Step 3: Build
echo "Building React app..."
npm run build

# Step 4: CD back
cd ..

# Step 5: Clean old build
echo "Cleaning old build..."
rm -rf $STATIC_DIR

# Step 6: Move new build
echo "Moving new build..."
mkdir -p $STATIC_DIR
mv $ANNOTATOR_DIR/build/* $STATIC_DIR/

# Step 7: Restore development config
# echo "Restoring development config..."
# cd $ANNOTATOR_DIR
# mv ${CONFIG_FILE}.bak $CONFIG_FILE

echo "Deployment complete!"
