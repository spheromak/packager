## Packager
packager is a simple app that will take a dockerfile and pass it through Golangs text template. With some added functions to help things along.

It exposes a struct of properties that can be used in the template used to build packages with fpm across multiple os's with the same basic dockerfile

### Examples
see examples directory 

### Template variables:
* `Version` - The version you asked to be installed
* `Rev` - The revision you supplied
* `OS` - The current OS/platform being built 


### Template Functions
* `makeList` - function that will take arguments and return them so you can range over them someplace else in your template

    {{ $foo := makeList "thing" "otherthing" "morethings" }} 
    {{ range $foo }}
     the thing here is: {{.}}
    {{endll}}
