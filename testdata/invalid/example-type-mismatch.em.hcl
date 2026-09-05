# The example is a string, but the field type is declared as an Int.
bounded_context "example" {
  title = "Example"

  field_type "pet_id" {
    type    = "Int"
    example = "not-a-number"
  }
}
