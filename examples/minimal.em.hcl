# Minimal Event Modeling document in native HCL.
# Block labels represent the JSON `id` or `name` identity fields.

slice "add-pet" {
  title      = "Add Pet"
  slice_type = "STATE_CHANGE"

  command "add-pet" {
    title = "Add Pet"
    type  = "COMMAND"

    dependency "evt-pet-001" {
      type         = "OUTBOUND"
      title        = "Pet Added"
      element_type = "EVENT"
    }
  }

  event "evt-pet-001" {
    title = "Pet Added"
    type  = "EVENT"

    field "pet_id" {
      type         = "Int"
      example      = 5
      id_attribute = true
    }
  }
}
