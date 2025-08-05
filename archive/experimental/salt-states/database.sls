database_packages:
  pkg.installed:
    - pkgs:
      - sqlite

database_config:
  file.managed:
    - name: /tmp/database-config.txt
    - contents: "Database configuration for {{ grains['id'] }}"
    - mode: 644

database_test:
  cmd.run:
    - name: echo "Database service configured on $(hostname)"
