# Kingsguard
A lightweight golang http request validator, for validating that request parameters match predefined expected schemas
kingsguard validates the http request itself, extracting the params and validating them against a schema
kingsguard returns errors specific to missing params

## Install
```
go get -u github.com/geniusmartinss/kingsguard
```

## Usage
Say you had a simple server with a handler (your kingdom)
```
import "github.com/geniusmartinss/kingsguard"

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":9000", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
    //Do whatever you want with the request
}

```
Now lets assume you wanted every visitor to come with certain params to gain entrance
you put jamie lannister at the gate with specific requirements
```
	jamie := kingsguard.Lannister{
		"fruit",
		true,
		"string",
		"p([a-z]+)ch",
		-1,
		10,
		"query",
	}

```
Meaning each visitor must come with a "fruit"
This fruit is required
The fruit must be of type string
There is no minimum length for the fruit
the maximum length for the fruit is 10
the fruit must match the pattern "p([a-z]+)ch"
the fruit must be sent as a query param

putting this all together we have
```
import "github.com/geniusmartinss/kingsguard"

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":9000", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	jamie := kingsguard.Lannister{
		"fruit",
		true,
		"string",
		"p([a-z]+)ch",
		-1,
		10,
		"query",
	}

	_,err := kingsguard.ValidateRequest(r, jamie)
	if err != nil {
	    	fmt.Fprint(w, err)
	}
}
```
And just like that, any request not matching the specification gets bounced.
Simple and sweet, just like peach :)

##Functionality
A schema
```
type Lannister struct {
	Field     string    //defines the name of the field you want to check for
	Required  bool      //defines if the field is required
	Datatype  string    //defines the required datatype for the value
	Regexp    string    //defines the pattern for which the value must match, empty string if no patter
	Min int       //for strings defines the minimum length of characters, for int defines min value -1 if no preferences, ignored for bool
	Max int       //for strings defines the maximum length of characters, for int defines max value -1 if no preferences, ignored for bool
	ParamType string    //defines if the parameters should be checked in "query" params or request "body"
}

```

## Contributing
Just Make a pull request :) No Dragons though

# Author
https://twitter.com/geniusmartins

## License
Released under the <a href ="http://www.opensource.org/licenses/MIT">MIT License</a>