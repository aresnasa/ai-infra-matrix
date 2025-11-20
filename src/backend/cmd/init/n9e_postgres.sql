

-- drop database if exists n9e_v6;



CREATE TABLE "users" (
    "id" bigserial,
    "username" varchar(64) not null,
    "nickname" varchar(64) not null,
    "password" varchar(128) not null default '',
    "phone" varchar(16) not null default '',
    "email" varchar(64) not null default '',
    "portrait" varchar(255) not null default '',
    "roles" varchar(255) not null,
    "contacts" varchar(1024),
    "maintainer" smallint not null default 0,
    "belong" varchar(191) DEFAULT '',
    "last_active_time" bigint DEFAULT 0,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id"),
    UNIQUE ("username")
);

insert into "users"(id, username, nickname, password, roles, create_at, create_by, update_at, update_by) values(1, 'root', '超管', 'root.2020', 'Admin', EXTRACT(EPOCH FROM NOW())::bigint, 'system', EXTRACT(EPOCH FROM NOW())::bigint, 'system');

CREATE TABLE "user_group" (
    "id" bigserial,
    "name" varchar(128) not null default '',
    "note" varchar(255) not null default '',
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id"));

insert into user_group(id, name, create_at, create_by, update_at, update_by) values(1, 'demo-root-group', EXTRACT(EPOCH FROM NOW())::bigint, 'root', EXTRACT(EPOCH FROM NOW())::bigint, 'root');

CREATE TABLE "user_group_member" (
    "id" bigserial,
    "group_id" bigint not null,
    "user_id" bigint not null,
    PRIMARY KEY("id")
);

insert into user_group_member(group_id, user_id) values(1, 1);

CREATE TABLE "configs" (
    "id" bigserial,
    "ckey" varchar(191) not null,
    "note" varchar(1024) NOT NULL DEFAULT '',
    "cval" text,
    "external"  bigint DEFAULT 0,
    "encrypted" bigint DEFAULT 0,
    "create_at" bigint DEFAULT 0,
    "create_by" varchar(64) NOT NULL DEFAULT '',
    "update_at" bigint DEFAULT 0,
    "update_by" varchar(64) NOT NULL DEFAULT '',
    PRIMARY KEY ("id")
);

CREATE TABLE "role" (
    "id" bigserial,
    "name" varchar(191) not null default '',
    "note" varchar(255) not null default '',
    PRIMARY KEY ("id"),
    UNIQUE ("name")
);

insert into "role"(name, note) values('Admin', 'Administrator role');
insert into "role"(name, note) values('Standard', 'Ordinary user role');
insert into "role"(name, note) values('Guest', 'Readonly user role');

CREATE TABLE "role_operation"(
    "id" bigserial,
    "role_name" varchar(128) not null,
    "operation" varchar(191) not null,
    PRIMARY KEY("id")
);

-- Admin is special, who has no concrete operation but can do anything.
insert into "role_operation"(role_name, operation) values('Guest', '/metric/explorer');
insert into "role_operation"(role_name, operation) values('Guest', '/object/explorer');
insert into "role_operation"(role_name, operation) values('Guest', '/log/explorer');
insert into "role_operation"(role_name, operation) values('Guest', '/trace/explorer');
insert into "role_operation"(role_name, operation) values('Guest', '/help/version');
insert into "role_operation"(role_name, operation) values('Guest', '/help/contact');

insert into "role_operation"(role_name, operation) values('Standard', '/metric/explorer');
insert into "role_operation"(role_name, operation) values('Standard', '/object/explorer');
insert into "role_operation"(role_name, operation) values('Standard', '/log/explorer');
insert into "role_operation"(role_name, operation) values('Standard', '/trace/explorer');
insert into "role_operation"(role_name, operation) values('Standard', '/help/version');
insert into "role_operation"(role_name, operation) values('Standard', '/help/contact');
insert into "role_operation"(role_name, operation) values('Standard', '/help/servers');
insert into "role_operation"(role_name, operation) values('Standard', '/help/migrate');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-rules-built-in');
insert into "role_operation"(role_name, operation) values('Standard', '/dashboards-built-in');
insert into "role_operation"(role_name, operation) values('Standard', '/trace/dependencies');
insert into "role_operation"(role_name, operation) values('Standard', '/users');
insert into "role_operation"(role_name, operation) values('Standard', '/user-groups');
insert into "role_operation"(role_name, operation) values('Standard', '/user-groups/add');
insert into "role_operation"(role_name, operation) values('Standard', '/user-groups/put');
insert into "role_operation"(role_name, operation) values('Standard', '/user-groups/del');
insert into "role_operation"(role_name, operation) values('Standard', '/busi-groups');
insert into "role_operation"(role_name, operation) values('Standard', '/busi-groups/add');
insert into "role_operation"(role_name, operation) values('Standard', '/busi-groups/put');
insert into "role_operation"(role_name, operation) values('Standard', '/busi-groups/del');
insert into "role_operation"(role_name, operation) values('Standard', '/targets');
insert into "role_operation"(role_name, operation) values('Standard', '/targets/add');
insert into "role_operation"(role_name, operation) values('Standard', '/targets/put');
insert into "role_operation"(role_name, operation) values('Standard', '/targets/del');
insert into "role_operation"(role_name, operation) values('Standard', '/dashboards');
insert into "role_operation"(role_name, operation) values('Standard', '/dashboards/add');
insert into "role_operation"(role_name, operation) values('Standard', '/dashboards/put');
insert into "role_operation"(role_name, operation) values('Standard', '/dashboards/del');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-rules');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-rules/add');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-rules/put');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-rules/del');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-mutes');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-mutes/add');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-mutes/del');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-subscribes');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-subscribes/add');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-subscribes/put');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-subscribes/del');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-cur-events');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-cur-events/del');
insert into "role_operation"(role_name, operation) values('Standard', '/alert-his-events');
insert into "role_operation"(role_name, operation) values('Standard', '/job-tpls');
insert into "role_operation"(role_name, operation) values('Standard', '/job-tpls/add');
insert into "role_operation"(role_name, operation) values('Standard', '/job-tpls/put');
insert into "role_operation"(role_name, operation) values('Standard', '/job-tpls/del');
insert into "role_operation"(role_name, operation) values('Standard', '/job-tasks');
insert into "role_operation"(role_name, operation) values('Standard', '/job-tasks/add');
insert into "role_operation"(role_name, operation) values('Standard', '/job-tasks/put');
insert into "role_operation"(role_name, operation) values('Standard', '/recording-rules');
insert into "role_operation"(role_name, operation) values('Standard', '/recording-rules/add');
insert into "role_operation"(role_name, operation) values('Standard', '/recording-rules/put');
insert into "role_operation"(role_name, operation) values('Standard', '/recording-rules/del');

-- for alert_rule | collect_rule | mute | dashboard grouping
CREATE TABLE "busi_group" (
    "id" bigserial,
    "name" varchar(191) not null,
    "label_enable" smallint not null default 0,
    "label_value" varchar(191) not null default '',
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id"),
    UNIQUE ("name")
);

insert into busi_group(id, name, create_at, create_by, update_at, update_by) values(1, 'Default Busi Group', EXTRACT(EPOCH FROM NOW())::bigint, 'root', EXTRACT(EPOCH FROM NOW())::bigint, 'root');

CREATE TABLE "busi_group_member" (
    "id" bigserial,
    "busi_group_id" bigint not null,
    "user_group_id" bigint not null,
    "perm_flag" char(2) not null,
    PRIMARY KEY ("id"));

insert into busi_group_member(busi_group_id, user_group_id, perm_flag) values(1, 1, 'rw');

-- for dashboard new version
CREATE TABLE "board" (
    "id" bigserial,
    "group_id" bigint not null default 0,
    "name" varchar(191) not null,
    "ident" varchar(200) not null default '',
    "tags" varchar(255) not null,
    "public" smallint not null default 0,
    "built_in" smallint not null default 0,
    "hide" smallint not null default 0,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    "note" varchar(1024) not null default '',
    "public_cate" bigint NOT NULL NOT NULL DEFAULT 0,
    PRIMARY KEY ("id"),
    UNIQUE ("group_id", "name"),
    KEY("ident")
);

-- for dashboard new version
CREATE TABLE "board_payload" (
    "id" bigint not null,
    "payload" text not null,
    UNIQUE ("id")
);

-- deprecated
CREATE TABLE "dashboard" (
    "id" bigserial,
    "group_id" bigint not null default 0,
    "name" varchar(191) not null,
    "tags" varchar(255) not null,
    "configs" varchar(8192),
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id"),
    UNIQUE ("group_id", "name")
);

-- deprecated
-- auto create the first subclass 'Default chart group' of dashboard
CREATE TABLE "chart_group" (
    "id" bigserial,
    "dashboard_id" bigint not null,
    "name" varchar(255) not null,
    "weight" int not null default 0,
    PRIMARY KEY ("id"));

-- deprecated
CREATE TABLE "chart" (
    "id" bigserial,
    "group_id" bigint not null,
    "configs" text,
    "weight" int not null default 0,
    PRIMARY KEY ("id"));

CREATE TABLE "chart_share" (
    "id" bigserial,
    "cluster" varchar(128) not null,
    "datasource_id" bigint NOT NULL NOT NULL DEFAULT 0,
    "configs" text,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    primary key ("id"));

CREATE TABLE "alert_rule" (
    "id" bigserial,
    "group_id" bigint not null default 0,
    "cate" varchar(128) not null,
    "datasource_ids" varchar(255) not null default '',
    "cluster" varchar(128) not null,
    "name" varchar(255) not null,
    "note" varchar(1024) not null default '',
    "prod" varchar(255) not null default '',
    "algorithm" varchar(255) not null default '',
    "algo_params" varchar(255),
    "delay" int not null default 0,
    "severity" smallint not null,
    "disabled" smallint not null,
    "prom_for_duration" int not null,
    "rule_config" text not null,
    "prom_ql" text not null,
    "prom_eval_interval" int not null,
    "enable_stime" varchar(255) not null default '00:00',
    "enable_etime" varchar(255) not null default '23:59',
    "enable_days_of_week" varchar(255) not null default '',
    "enable_in_bg" smallint not null default 0,
    "notify_recovered" smallint not null,
    "notify_channels" varchar(255) not null default '',
    "notify_groups" varchar(255) not null default '',
    "notify_repeat_step" int not null default 0,
    "notify_max_number" int not null default 0,
    "recover_duration" int not null default 0,
    "callbacks" varchar(4096) not null default '',
    "runbook_url" varchar(4096),
    "append_tags" varchar(255) not null default '',
    "annotations" text not null,
    "extra_config" text,
    "notify_rule_ids" varchar(1024) DEFAULT '',
    "notify_version" int DEFAULT 0,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    "cron_pattern" varchar(64),
    "datasource_queries" text,
    PRIMARY KEY ("id"));

CREATE TABLE "alert_mute" (
    "id" bigserial,
    "group_id" bigint not null default 0,
    "prod" varchar(255) not null default '',
    "note" varchar(1024) not null default '',
    "cate" varchar(128) not null,
    "cluster" varchar(128) not null,
    "datasource_ids" varchar(255) not null default '',
    "tags" varchar(4096) default '[]',
    "cause" varchar(255) not null default '',
    "btime" bigint not null default 0,
    "etime" bigint not null default 0,
    "disabled" smallint not null default 0,
    "mute_time_type" smallint not null default 0,
    "periodic_mutes" varchar(4096) not null default '',
    "severities" varchar(32) not null default '',
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id"));

CREATE TABLE "alert_subscribe" (
    "id" bigserial,
    "name" varchar(255) not null default '',
    "disabled" smallint not null default 0,
    "group_id" bigint not null default 0,
    "prod" varchar(255) not null default '',
    "cate" varchar(128) not null,
    "datasource_ids" varchar(255) not null default '',
    "cluster" varchar(128) not null,
    "rule_id" bigint not null default 0,
    "rule_ids" varchar(1024),
    "severities" varchar(32) not null default '',
    "tags" varchar(4096) not null default '',
    "redefine_severity" smallint default 0,
    "new_severity" smallint not null,
    "redefine_channels" smallint default 0,
    "new_channels" varchar(255) not null default '',
    "user_group_ids" varchar(250) not null,
    "busi_groups" varchar(4096),
    "note" VARCHAR(1024) DEFAULT '',
    "webhooks" text not null,
    "extra_config" text,
    "redefine_webhooks" smallint default 0,
    "for_duration" bigint not null default 0,
    "notify_rule_ids" varchar(1024) DEFAULT '',
    "notify_version" int DEFAULT 0,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id"));

CREATE TABLE "target" (
    "id" bigserial,
    "group_id" bigint not null default 0,
    "ident" varchar(191) not null,
    "note" varchar(255) not null default '',
    "tags" varchar(512) not null default '',
    "host_tags" text,
    "host_ip" varchar(15) default '',
    "agent_version" varchar(255) default '',
    "engine_name" varchar(255) DEFAULT '',
    "os" VARCHAR(31) DEFAULT '',
    "update_at" bigint not null default 0,
    PRIMARY KEY ("id"),
    UNIQUE ("ident"));


CREATE TABLE "metric_view" (
    "id" bigserial,
    "name" varchar(191) not null default '',
    "cate" smallint not null,
    "configs" varchar(8192) not null default '',
    "create_at" bigint not null default 0,
    "create_by" bigint not null default 0,
    "update_at" bigint not null default 0,
    PRIMARY KEY ("id"));

insert into metric_view(name, cate, configs) values('Host View', 0, '{"filters":[{"oper":"=","label":"__name__","value":"cpu_usage_idle"}],"dynamicLabels":[],"dimensionLabels":[{"label":"ident","value":""}]}');

CREATE TABLE "recording_rule" (
    "id" bigserial,
    "group_id" bigint not null default '0',
    "datasource_ids" varchar(255) not null default '',
    "cluster" varchar(128) not null,
    "name" varchar(255) not null,
    "note" varchar(255) not null,
    "disabled" smallint not null default 0,
    "prom_ql" varchar(8192) not null,
    "prom_eval_interval" int not null,
    "cron_pattern" varchar(255) default '',
    "append_tags" varchar(255) default '',
    "query_configs" text NOT NULL,
    "create_at" bigint default '0',
    "create_by" varchar(64) default '',
    "update_at" bigint default '0',
    "update_by" varchar(64) default '',
    "datasource_queries" text,
    PRIMARY KEY ("id"));

CREATE TABLE "alert_aggr_view" (
    "id" bigserial,
    "name" varchar(191) not null default '',
    "rule" varchar(2048) not null default '',
    "cate" smallint not null,
    "create_at" bigint not null default 0,
    "create_by" bigint not null default 0,
    "update_at" bigint not null default 0,
    PRIMARY KEY ("id"));

insert into alert_aggr_view(name, rule, cate) values('By BusiGroup, Severity', 'field:group_name::field:severity', 0);
insert into alert_aggr_view(name, rule, cate) values('By RuleName', 'field:rule_name', 0);

CREATE TABLE "alert_cur_event" (
    "id" bigint not null,
    "cate" varchar(128) not null,
    "datasource_id" bigint not null default 0,
    "cluster" varchar(128) not null,
    "group_id" bigint not null,
    "group_name" varchar(255) not null default '',
    "hash" varchar(64) not null,
    "rule_id" bigint not null,
    "rule_name" varchar(255) not null,
    "rule_note" varchar(2048) not null default 'alert rule note',
    "rule_prod" varchar(255) not null default '',
    "rule_algo" varchar(255) not null default '',
    "severity" smallint not null,
    "prom_for_duration" int not null,
    "prom_ql" varchar(8192) not null,
    "prom_eval_interval" int not null,
    "callbacks" varchar(2048) not null default '',
    "runbook_url" varchar(255),
    "notify_recovered" smallint not null,
    "notify_channels" varchar(255) not null default '',
    "notify_groups" varchar(255) not null default '',
    "notify_repeat_next" bigint not null default 0,
    "notify_cur_number" int not null default 0,
    "target_ident" varchar(191) not null default '',
    "target_note" varchar(191) not null default '',
    "first_trigger_time" bigint,
    "trigger_time" bigint not null,
    "trigger_value" text not null,
    "annotations" text not null,
    "rule_config" text not null,
    "tags" varchar(1024) not null default '',
    "original_tags" text,
    "notify_rule_ids" text,
    PRIMARY KEY ("id"));

CREATE TABLE "alert_his_event" (
    "id" bigint not null AUTO_INCREMENT,
    "is_recovered" smallint not null,
    "cate" varchar(128) not null,
    "datasource_id" bigint not null default 0,
    "cluster" varchar(128) not null,
    "group_id" bigint not null,
    "group_name" varchar(255) not null default '',
    "hash" varchar(64) not null,
    "rule_id" bigint not null,
    "rule_name" varchar(255) not null,
    "rule_note" varchar(2048) not null default 'alert rule note',
    "rule_prod" varchar(255) not null default '',
    "rule_algo" varchar(255) not null default '',
    "severity" smallint not null,
    "prom_for_duration" int not null,
    "prom_ql" varchar(8192) not null,
    "prom_eval_interval" int not null,
    "callbacks" varchar(2048) not null default '',
    "runbook_url" varchar(255),
    "notify_recovered" smallint not null,
    "notify_channels" varchar(255) not null default '',
    "notify_groups" varchar(255) not null default '',
    "notify_cur_number" int not null default 0,
    "target_ident" varchar(191) not null default '',
    "target_note" varchar(191) not null default '',
    "first_trigger_time" bigint,
    "trigger_time" bigint not null,
    "trigger_value" text not null,
    "recover_time" bigint not null default 0,
    "last_eval_time" bigint not null default 0,
    "tags" varchar(1024) not null default '',
    "original_tags" text,
    "annotations" text not null,
    "rule_config" text not null,
    "notify_rule_ids" text,
    PRIMARY KEY ("id"));

CREATE TABLE "board_busigroup" (
  "busi_group_id" bigint(20) NOT NULL DEFAULT '0',
  "board_id" bigint(20) NOT NULL DEFAULT '0',
  PRIMARY KEY ("busi_group_id", "board_id")
);

CREATE TABLE "builtin_components" (
  "id" bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  "ident" varchar(191) NOT NULL,
  "logo" text'logo of component''',
  "readme" text NOT NULL'readme of component''',
  "created_at" bigint NOT NULL DEFAULT 0'create time''',
  "created_by" varchar(191) NOT NULL DEFAULT '''creator''',
  "updated_at" bigint NOT NULL DEFAULT 0'update time''',
  "updated_by" varchar(191) NOT NULL DEFAULT '''updater''',
  "disabled" int NOT NULL DEFAULT 0'is disabled or not''',
  PRIMARY KEY ("id"));

CREATE TABLE "builtin_payloads" (
  "id" bigint(20) NOT NULL AUTO_INCREMENT'unique identifier''',
  "component_id" bigint NOT NULL DEFAULT 0'component_id of payload''',
  "uuid" bigint(20) NOT NULL'uuid of payload''',
  "type" varchar(191) NOT NULL'type of payload''',
  "component" varchar(191) NOT NULL'component of payload''',
  "cate" varchar(191) NOT NULL'category of payload''',
  "name" varchar(191) NOT NULL'name of payload''',
  "tags" varchar(191) NOT NULL DEFAULT '''tags of payload''',
  "content" text NOT NULL'content of payload''',
  "note" varchar(1024) NOT NULL DEFAULT '''note of payload''',
  "created_at" bigint(20) NOT NULL DEFAULT 0'create time''',
  "created_by" varchar(191) NOT NULL DEFAULT '''creator''',
  "updated_at" bigint(20) NOT NULL DEFAULT 0'update time''',
  "updated_by" varchar(191) NOT NULL DEFAULT '''updater''',
  PRIMARY KEY ("id"));

CREATE TABLE notification_record (
    "id" BIGINT PRIMARY KEY AUTO_INCREMENT,
    "notify_rule_id" BIGINT NOT NULL DEFAULT 0,
    "event_id"  bigint NOT NULL,
    "sub_id"  bigint,
    "channel" varchar(255) NOT NULL,
    "status" bigint,
    "target" varchar(1024) NOT NULL,
    "details" varchar(2048) DEFAULT '',
    "created_at" bigint NOT NULL);

CREATE TABLE "task_tpl"
(
    "id"        int NOT NULL AUTO_INCREMENT,
    "group_id"  int not null,
    "title"     varchar(255) not null default '',
    "account"   varchar(64)  not null,
    "batch"     int not null default 0,
    "tolerance" int not null default 0,
    "timeout"   int not null default 0,
    "pause"     varchar(255) not null default '',
    "script"    text         not null,
    "args"      varchar(512) not null default '',
    "tags"      varchar(255) not null default '',
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id"));

CREATE TABLE "task_tpl_host"
(
    "ii"   int NOT NULL AUTO_INCREMENT,
    "id"   int not null,
    "host" varchar(128)  not null,
    PRIMARY KEY ("ii"));

CREATE TABLE "task_record"
(
    "id" bigint not null,
    "event_id" bigint not null default 0,
    "group_id" bigint not null,
    "ibex_address"   varchar(128) not null,
    "ibex_auth_user" varchar(128) not null default '',
    "ibex_auth_pass" varchar(128) not null default '',
    "title"     varchar(255)    not null default '',
    "account"   varchar(64)     not null,
    "batch"     int    not null default 0,
    "tolerance" int    not null default 0,
    "timeout"   int    not null default 0,
    "pause"     varchar(255)    not null default '',
    "script"    text            not null,
    "args"      varchar(512)    not null default '',
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    PRIMARY KEY ("id"));

CREATE TABLE "alerting_engines"
(
    "id" int NOT NULL AUTO_INCREMENT,
    "instance" varchar(128) not null default '',
    "datasource_id" bigint not null default 0,
    "engine_cluster" varchar(128) not null default '',
    "clock" bigint not null,
    PRIMARY KEY ("id"));

CREATE TABLE "datasource"
(
    "id" int NOT NULL AUTO_INCREMENT,
    "name" varchar(191) not null default '',
    "identifier" varchar(255) not null default '',
    "description" varchar(255) not null default '',
    "category" varchar(255) not null default '',
    "plugin_id" int not null default 0,
    "plugin_type" varchar(255) not null default '',
    "plugin_type_name" varchar(255) not null default '',
    "cluster_name" varchar(255) not null default '',
    "settings" text not null,
    "status" varchar(255) not null default '',
    "http" varchar(4096) not null default '',
    "auth" varchar(8192) not null default '',
    "is_default" boolean,
    "created_at" bigint not null default 0,
    "created_by" varchar(64) not null default '',
    "updated_at" bigint not null default 0,
    "updated_by" varchar(64) not null default '',
    UNIQUE ("name"),
    PRIMARY KEY ("id")
);

CREATE TABLE "builtin_cate" (
    "id" bigserial,
    "name" varchar(191) not null,
    "user_id" bigint not null default 0,
    PRIMARY KEY ("id")
);

CREATE TABLE "notify_tpl" (
    "id" bigserial,
    "channel" varchar(32) not null,
    "name" varchar(255) not null,
    "content" text not null,
    "create_at" bigint DEFAULT 0,
    "create_by" varchar(64) DEFAULT '',
    "update_at" bigint DEFAULT 0,
    "update_by" varchar(64) DEFAULT '',
    PRIMARY KEY ("id"),
    UNIQUE ("channel")
);

CREATE TABLE "sso_config" (
    "id" bigserial,
    "name" varchar(191) not null,
    "content" text not null,
    "update_at" bigint DEFAULT 0,
    PRIMARY KEY ("id"),
    UNIQUE ("name")
);

CREATE TABLE "es_index_pattern" (
    "id" bigserial,
    "datasource_id" bigint not null default 0,
    "name" varchar(191) not null,
    "time_field" varchar(128) not null default '@timestamp',
    "allow_hide_system_indices" smallint not null default 0,
    "fields_format" varchar(4096) not null default '',
    "cross_cluster_enabled" int not null default 0,
    "note" varchar(1024) not null default '',
    "create_at" bigint default '0',
    "create_by" varchar(64) default '',
    "update_at" bigint default '0',
    "update_by" varchar(64) default '',
    PRIMARY KEY ("id"),
    UNIQUE ("datasource_id", "name")
);


CREATE TABLE "builtin_metrics" (
    "id" bigint NOT NULL AUTO_INCREMENT,
    "collector" varchar(191) NOT NULL'type of collector''',
    "typ" varchar(191) NOT NULL'type of metric''',
    "name" varchar(191) NOT NULL'name of metric''',
    "unit" varchar(191) NOT NULL'unit of metric''',
    "lang" varchar(191) NOT NULL DEFAULT 'zh''language''',
    "note" varchar(4096) NOT NULL'description of metric''',
    "expression" varchar(4096) NOT NULL'expression of metric''',
    "created_at" bigint NOT NULL DEFAULT 0'create time''',
    "created_by" varchar(191) NOT NULL DEFAULT '''creator''',
    "updated_at" bigint NOT NULL DEFAULT 0'update time''',
    "updated_by" varchar(191) NOT NULL DEFAULT '''updater''',
    "uuid" bigint NOT NULL DEFAULT 0'uuid''',
    PRIMARY KEY ("id"));

CREATE TABLE "metric_filter" (
  "id" bigint NOT NULL AUTO_INCREMENT,
  "name"  varchar(191) NOT NULL'name of metric filter''',
  "configs"  varchar(4096) NOT NULL'configuration of metric filter''',
  "groups_perm" text,
  "create_at" bigint NOT NULL DEFAULT 0'create time''',
  "create_by" varchar(191) NOT NULL DEFAULT '''creator''',
  "update_at" bigint NOT NULL DEFAULT 0'update time''',
  "update_by" varchar(191) NOT NULL DEFAULT '''updater''',
  PRIMARY KEY ("id"));

CREATE TABLE "target_busi_group" (
  "id" bigint NOT NULL AUTO_INCREMENT,
  "target_ident" varchar(191) NOT NULL,
  "group_id" bigint NOT NULL,
  "update_at" bigint NOT NULL,
  PRIMARY KEY ("id"),
    UNIQUE ("target_ident","group_id"));


CREATE TABLE "dash_annotation" (
    "id" bigserial,
    "dashboard_id" bigint not null,
    "panel_id" varchar(191) not null,
    "tags" text,
    "description" text,
    "config" text,
    "time_start" bigint not null default 0,
    "time_end" bigint not null default 0,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id"));

CREATE TABLE "user_token" (
    "id" bigint NOT NULL AUTO_INCREMENT,
    "username" varchar(255) NOT NULL DEFAULT '',
    "token_name" varchar(255) NOT NULL DEFAULT '',
    "token" varchar(255) NOT NULL DEFAULT '',
    "create_at" bigint NOT NULL DEFAULT 0,
    "last_used" bigint NOT NULL DEFAULT 0,
    PRIMARY KEY ("id")
);


CREATE TABLE "notify_rule" (
    "id" bigserial,
    "name" varchar(255) not null,
    "description" text,
    "enable" smallint not null default 0,
    "user_group_ids" varchar(255) not null default '',
    "notify_configs" text,
    "pipeline_configs" text,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id")
);

CREATE TABLE "notify_channel" (
    "id" bigserial,
    "name" varchar(255) not null,
    "ident" varchar(255) not null,
    "description" text, 
    "enable" smallint not null default 0,
    "param_config" text,
    "request_type" varchar(50) not null,
    "request_config" text,
    "weight" int not null default 0,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id")
);

CREATE TABLE "message_template" (
    "id" bigserial,
    "name" varchar(64) not null,
    "ident" varchar(64) not null,
    "content" text,
    "user_group_ids" varchar(64),
    "notify_channel_ident" varchar(64) not null default '',
    "private" int not null default 0,
    "weight" int not null default 0,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id")
);

CREATE TABLE "event_pipeline" (
    "id" bigserial,
    "name" varchar(128) not null,
    "team_ids" text,
    "description" varchar(255) not null default '',
    "filter_enable" smallint not null default 0,
    "label_filters" text,
    "attr_filters" text,
    "processor_configs" text,
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id")
);

CREATE TABLE "embedded_product" (
    "id" bigint NOT NULL AUTO_INCREMENT,
    "name" varchar(255) DEFAULT NULL,
    "url" varchar(255) DEFAULT NULL,
    "is_private" boolean DEFAULT NULL,
    "team_ids" varchar(255),
    "create_at" bigint not null default 0,
    "create_by" varchar(64) not null default '',
    "update_at" bigint not null default 0,
    "update_by" varchar(64) not null default '',
    PRIMARY KEY ("id")
);

CREATE TABLE "task_meta"
(
    "id"          bigint NOT NULL AUTO_INCREMENT,
    "title"       varchar(255)    not null default '',
    "account"     varchar(64)     not null,
    "batch"       bigint          not null default 0,
    "tolerance"   bigint          not null default 0,
    "timeout"     bigint    not null default 0,
    "pause"       varchar(255)    not null default '',
    "script"      text            not null,
    "args"        varchar(512)    not null default '',
    "stdin"       varchar(1024)   not null default '',
    "creator"     varchar(64)     not null default '',
    "created"     timestamp       not null default CURRENT_TIMESTAMP,
    PRIMARY KEY ("id")) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

/* start|cancel|kill|pause */
CREATE TABLE "task_action"
(
    "id"     bigint not null,
    "action" varchar(32)     not null,
    "clock"  bigint          not null default 0,
    PRIMARY KEY ("id")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE "task_scheduler"
(
    "id"        bigint not null,
    "scheduler" varchar(128)    not null default '') ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE "task_scheduler_health"
(
    "scheduler" varchar(128) NOT NULL,
    "clock"     bigint not null,
    UNIQUE ("scheduler")) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE "task_host_doing"
(
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "clock"  bigint          not null default 0,
    "action" varchar(16)     not null) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_0
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_1
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_2
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_3
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_4
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_5
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_6
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_7
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_8
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_9
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_10
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_11
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_12
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_13
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_14
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_15
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_16
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_17
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_18
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_19
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_20
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_21
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_22
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_23
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_24
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_25
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_26
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_27
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_28
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_29
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_30
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_31
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_32
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_33
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_34
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_35
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_36
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_37
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_38
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_39
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_40
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_41
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_42
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_43
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_44
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_45
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_46
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_47
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_48
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_49
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_50
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_51
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_52
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_53
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_54
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_55
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_56
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_57
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_58
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_59
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_60
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_61
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_62
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_63
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_64
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_65
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_66
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_67
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_68
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_69
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_70
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_71
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_72
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_73
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_74
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_75
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_76
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_77
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_78
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_79
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_80
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_81
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_82
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_83
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_84
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_85
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_86
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_87
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_88
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_89
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_90
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_91
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_92
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_93
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_94
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_95
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_96
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_97
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_98
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE task_host_99
(
    "ii"     bigint NOT NULL AUTO_INCREMENT,
    "id"     bigint not null,
    "host"   varchar(128)    not null,
    "status" varchar(32)     not null,
    "stdout" text,
    "stderr" text,
    UNIQUE ("id", "host"),
    PRIMARY KEY ("ii")
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE "source_token" (
    "id" bigint NOT NULL AUTO_INCREMENT,
    "source_type" varchar(64) NOT NULL DEFAULT '',
    "source_id" varchar(255) NOT NULL DEFAULT '',
    "token" varchar(255) NOT NULL DEFAULT '',
    "expire_at" bigint NOT NULL DEFAULT 0,
    "create_at" bigint NOT NULL DEFAULT 0,
    "create_by" varchar(64) NOT NULL DEFAULT '',
    PRIMARY KEY ("id"));


CREATE INDEX "idx_target_idx_host_ip" ON "target" ("host_ip");
CREATE INDEX "idx_target_idx_agent_version" ON "target" ("agent_version");
CREATE INDEX "idx_target_idx_engine_name" ON "target" ("engine_name");
CREATE INDEX "idx_target_idx_os" ON "target" ("os");
CREATE INDEX "idx_recording_rule_group_id" ON "recording_rule" ("group_id");
CREATE INDEX "idx_recording_rule_update_at" ON "recording_rule" ("update_at");
CREATE INDEX "idx_alert_his_event_idx_last_eval_time" ON "alert_his_event" ("last_eval_time");
CREATE INDEX "idx_builtin_payloads_idx_component" ON "builtin_payloads" ("component");
CREATE INDEX "idx_builtin_payloads_idx_name" ON "builtin_payloads" ("name");
CREATE INDEX "idx_builtin_payloads_idx_cate" ON "builtin_payloads" ("cate");
CREATE INDEX "idx_builtin_payloads_idx_uuid" ON "builtin_payloads" ("uuid");
CREATE INDEX "idx_builtin_payloads_idx_type" ON "builtin_payloads" ("type");
CREATE INDEX "idx_notification_record_idx_evt" ON "notification_record" (event_id);
CREATE INDEX "idx_task_record_idx_event_id" ON "task_record" ("event_id");
CREATE INDEX "idx_alerting_engines_idx_inst" ON "alerting_engines" ("instance");
CREATE INDEX "idx_alerting_engines_idx_clock" ON "alerting_engines" ("clock");
CREATE INDEX "idx_builtin_metrics_idx_uuid" ON "builtin_metrics" ("uuid");
CREATE INDEX "idx_builtin_metrics_idx_collector" ON "builtin_metrics" ("collector");
CREATE INDEX "idx_builtin_metrics_idx_typ" ON "builtin_metrics" ("typ");
CREATE INDEX "idx_builtin_metrics_idx_builtinmetric_name" ON "builtin_metrics" ("name" ASC);
CREATE INDEX "idx_builtin_metrics_idx_lang" ON "builtin_metrics" ("lang");
CREATE INDEX "idx_metric_filter_idx_metricfilter_name" ON "metric_filter" ("name" ASC);
CREATE INDEX "idx_dash_annotation_idx_dashboard_id" ON "dash_annotation" ("dashboard_id");
CREATE INDEX "idx_task_meta_idx_task_meta_creator" ON "task_meta" ("creator");
CREATE INDEX "idx_task_meta_idx_task_meta_created" ON "task_meta" ("created");
CREATE INDEX "idx_task_host_doing_idx_task_host_doing_id" ON "task_host_doing" ("id");
CREATE INDEX "idx_task_host_doing_idx_task_host_doing_host" ON "task_host_doing" ("host");
CREATE INDEX "idx_source_token_idx_source_type_id_token" ON "source_token" ("source_type", "source_id", "token");