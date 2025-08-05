common_packages:
  pkg.installed:
    - pkgs:
      - curl
      - git
      - vim

create_test_file:
  file.managed:
    - name: /tmp/salt-test.txt
    - contents: "Salt configuration applied successfully"
    - mode: 644

common_service_check:
  cmd.run:
    - name: echo "Common configuration applied on $(hostname)"
