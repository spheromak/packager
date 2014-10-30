Packager is a simple app that will take a dockerfile and pass it through Golangs text template. 

It exposes a struct of properties that can be used in the template used to build packages with fpm across multiple os's with the same basic dockerfile

see examples for more info

Template variables:
* `Version` - The version you asked to be installed
* `Rev` - The revision you supplied
* `OS` - The current OS/platform being built 

