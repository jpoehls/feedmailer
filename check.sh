#!/bin/sh

gb build && go vet ./src/feedmailer && golint ./src/feedmailer && gb test