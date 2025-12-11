{
  "identities": [
    {
      "name": "admin",
      "credentials": [
        {
          "accessKey": "{{SEAWEEDFS_ACCESS_KEY}}",
          "secretKey": "{{SEAWEEDFS_SECRET_KEY}}"
        }
      ],
      "actions": [
        "Admin",
        "Read",
        "Write",
        "List",
        "Tagging"
      ]
    },
    {
      "name": "app",
      "credentials": [
        {
          "accessKey": "{{SEAWEEDFS_APP_ACCESS_KEY}}",
          "secretKey": "{{SEAWEEDFS_APP_SECRET_KEY}}"
        }
      ],
      "actions": [
        "Read",
        "Write",
        "List",
        "Tagging"
      ]
    },
    {
      "name": "readonly",
      "credentials": [
        {
          "accessKey": "{{SEAWEEDFS_READONLY_ACCESS_KEY}}",
          "secretKey": "{{SEAWEEDFS_READONLY_SECRET_KEY}}"
        }
      ],
      "actions": [
        "Read",
        "List"
      ]
    }
  ]
}
