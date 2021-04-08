# pankul

A database helper.

Helps to quickly create and edit flaarum tables


## Projects Used

* Golang
* Ubuntu
* [Flaarum](https://github.com/bankole7782/flaarum)


## Setup

### Users Table Setup

Note that user creation and updating is not part of this project. You as the administrator is to provide this. This is to
ensure that you can use any form of authentication you want eg. social auth (Facebook, google, twitter), passwords,
fingerprint, keys etc.

Create a users table with the following properties:

1. it must also have fields `firstname` and `surname` for easy recognition.
2. it must also have field `email` for communications.
3. it must also have field `timezone` for datetime data. Example value is 'Africa/Lagos'

You must also provide a function that would get the currently logged in users. The function is given the request object
to get the cookies for its purpose. Set the `pankul.GetCurrentUser` to this function. The function has the following
declaration `func(r *http.Request) (int64, error)`.

The `pankul.GetCurrentUser` should return 0 for public.


### Begin

Get the framework through the following command `go get -u github.com/bankole7782/pankul`

There is a sample application which details how to complete the setup. Take a look at it [here](https://github.com/bankole7782/pankul/tree/master/pankul_sample)

Copy the folder `pankul_files` from the main repo into the same path as your `main.go`.

Make sure you look at `main.go` in the sample app, copy and edit it to your own preferences.

Go to `/pankul/setup/` to create some tables that the project would need.

Then go to `/pankul/page/` to start using this project.



### Theming Your Project

The sample project has no design. To make it beautiful make a template from this template :`pankul_files/bad-base.html`
. Save it to your project and then point your version to `pankul.BaseTemplate`.

Also if you want to add dynamic contents to any `pankul` page, please use JavaScript.
First check the address of the page `window.location` before adding it.


## FAQs
### When is X Database Support Coming

I don't intend to support more than one database so has to make the work cheaper.


## License

Released with the MIT License
