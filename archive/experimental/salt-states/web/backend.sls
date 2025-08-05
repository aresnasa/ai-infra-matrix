backend_packages:
  pkg.installed:
    - pkgs:
      - python3

backend_config:
  file.managed:
    - name: /tmp/backend-config.txt
    - contents: "Backend configuration for {{ grains['id'] }}"
    - mode: 644

backend_test:
  cmd.run:
    - name: echo "Backend service configured on $(hostname)"
