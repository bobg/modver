module foo.bar/a

go 1.25

require (
	foo.bar/d/v3 v3.0.0
)

replace foo.bar/d/v3 => ../d2
