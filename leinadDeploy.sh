#!/bin/bash

# Move to code repo
echo "Beginning deployment of Leinad website..."
cd ../../leinad/code

# Pull in any new code
echo "Pulling changes locally..."
git pull

# Run npm install to check for new packages
echo "Installing new packages..."
npm install

# Build for production
echo "Building for production..."
npm run build

# Copy the dist files to the view page
echo "Deploying new build..."
cp -r ./dist/* ../../leinad/html/

echo "Deployment completed."
