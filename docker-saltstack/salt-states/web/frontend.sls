frontend_packages:
  pkg.installed:
    - pkgs:
      - nginx

frontend_config:
  file.managed:
    - name: /tmp/frontend-config.txt
    - contents: "Frontend configuration for {{ grains['id'] }}"
    - mode: 644

frontend_test:
  cmd.run:
    - name: echo "Frontend service configured on $(hostname)"
