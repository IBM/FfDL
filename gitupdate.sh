#!/bin/bash

CURRENTBRANCH=`git branch | sed -n '/\* /s///p'`
echo $CURRENTBRANCH
git fetch --all --prune
git checkout master
git merge --ff-only upstream/master
git push origin master
git checkout $CURRENTBRANCH
