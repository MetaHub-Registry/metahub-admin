#!/bin/bash


cat go.mod|awk '!/^replace/' > go.mod.tmp
mv go.mod.tmp go.mod