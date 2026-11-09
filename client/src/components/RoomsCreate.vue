<template>
  <v-card class="nr-create" role="dialog" aria-labelledby="nr-create-title">
    <header class="nr-create-head">
      <h2 id="nr-create-title">New room</h2>
      <span v-if="roomLimit > 0" class="nr-mono nr-create-quota">{{ roomCount }} of {{ roomLimit }} used</span>
      <span class="flex-grow-1" />
      <button type="button" class="nr-btn is-icon" aria-label="Close" @click="Close"><v-icon small>mdi-close</v-icon></button>
    </header>

    <div class="nr-create-body">
      <v-form ref="form" v-model="valid" lazy-validation @submit.prevent="CreateAndStart()">
        <div class="nr-templates" role="group" aria-label="Templates">
          <button
            v-for="t in templates"
            :key="t.key"
            type="button"
            class="nr-template"
            :class="{ 'is-on': template === t.key }"
            :aria-pressed="template === t.key"
            @click="useTemplate(t.key)"
          >
            <v-icon small>{{ t.icon }}</v-icon>
            <span>{{ t.name }}</span>
          </button>
        </div>

        <div class="nr-grid2">
          <div class="nr-field is-wide">
            <label class="nr-label" for="nr-c-title">Title</label>
            <v-text-field id="nr-c-title" v-model="meta.title" outlined dense hide-details="auto" placeholder="Movie night, Lo-fi beats, …" maxlength="80" autofocus />
          </div>

          <div class="nr-field">
            <label class="nr-label" for="nr-c-name">Name</label>
            <v-text-field
              id="nr-c-name"
              v-model="data.name"
              outlined
              dense
              hide-details="auto"
              class="nr-input-mono"
              :rules="[ rules.minLen(2), rules.containerNameStart, rules.containerName ]"
              autocomplete="off"
              placeholder="random if empty"
            />
            <span class="nr-mono nr-hint">{{ host }}/r/{{ data.name || '…' }}</span>
          </div>

          <div class="nr-field">
            <span class="nr-label">Visibility</span>
            <div class="nr-seg is-block is-field" role="group" aria-label="Visibility">
              <button
                v-for="v in visibilityItems"
                :key="v.value"
                type="button"
                :class="{ 'is-on': meta.visibility === v.value }"
                :disabled="v.disabled"
                :title="v.text"
                @click="meta.visibility = v.value"
              >{{ v.short }}</button>
            </div>
          </div>

          <div class="nr-field is-wide">
            <span class="nr-label">Browser</span>
            <div v-if="nekoImages.length <= 4" class="nr-engines" role="group" aria-label="Browser image">
              <button
                v-for="img in nekoImages"
                :key="img"
                type="button"
                class="nr-engine"
                :class="{ 'is-on': data.neko_image === img }"
                :aria-pressed="data.neko_image === img"
                :title="img"
                @click="data.neko_image = img"
              >
                <span class="nr-engine-name">{{ engineOf(img) }}</span>
                <span class="nr-mono nr-engine-image">{{ img }}</span>
              </button>
            </div>
            <v-select v-else v-model="data.neko_image" :items="nekoImages" outlined dense hide-details class="nr-input-mono" aria-label="Browser image" />
          </div>

          <div class="nr-field">
            <label class="nr-label" for="nr-c-user">Viewer password</label>
            <v-text-field
              id="nr-c-user"
              v-model="data.user_pass"
              outlined
              dense
              hide-details
              class="nr-input-mono"
              :type="showUserPass ? 'text' : 'password'"
              autocomplete="off"
              placeholder="auto"
            >
              <template v-slot:append>
                <button type="button" class="nr-btn is-icon is-sm" :aria-label="showUserPass ? 'Hide' : 'Show'" @click="showUserPass = !showUserPass"><v-icon small>{{ showUserPass ? 'mdi-eye-off-outline' : 'mdi-eye-outline' }}</v-icon></button>
                <button type="button" class="nr-btn is-icon is-sm" aria-label="Generate" @click="generate('user_pass')"><v-icon small>mdi-dice-5-outline</v-icon></button>
                <button type="button" class="nr-btn is-icon is-sm" :class="{ 'is-done': copied === 'user_pass' }" aria-label="Copy" :disabled="!data.user_pass" @click="copy('user_pass')"><v-icon small>{{ copied === 'user_pass' ? 'mdi-check' : 'mdi-content-copy' }}</v-icon></button>
              </template>
            </v-text-field>
          </div>

          <div class="nr-field">
            <label class="nr-label" for="nr-c-admin">Admin password</label>
            <v-text-field
              id="nr-c-admin"
              v-model="data.admin_pass"
              outlined
              dense
              hide-details
              class="nr-input-mono"
              :type="showAdminPass ? 'text' : 'password'"
              autocomplete="off"
              placeholder="auto"
            >
              <template v-slot:append>
                <button type="button" class="nr-btn is-icon is-sm" :aria-label="showAdminPass ? 'Hide' : 'Show'" @click="showAdminPass = !showAdminPass"><v-icon small>{{ showAdminPass ? 'mdi-eye-off-outline' : 'mdi-eye-outline' }}</v-icon></button>
                <button type="button" class="nr-btn is-icon is-sm" aria-label="Generate" @click="generate('admin_pass')"><v-icon small>mdi-dice-5-outline</v-icon></button>
                <button type="button" class="nr-btn is-icon is-sm" :class="{ 'is-done': copied === 'admin_pass' }" aria-label="Copy" :disabled="!data.admin_pass" @click="copy('admin_pass')"><v-icon small>{{ copied === 'admin_pass' ? 'mdi-check' : 'mdi-content-copy' }}</v-icon></button>
              </template>
            </v-text-field>
          </div>

          <div class="nr-field">
            <span class="nr-label">Screen</span>
            <v-select v-model="data.screen" :items="availableScreens" outlined dense hide-details class="nr-input-mono" aria-label="Screen" />
          </div>

          <div class="nr-field">
            <label class="nr-label" for="nr-c-start">Start page</label>
            <v-text-field id="nr-c-start" v-model="meta.start_url" outlined dense hide-details class="nr-input-mono" placeholder="https://www.youtube.com" clearable />
          </div>

          <div class="nr-field">
            <label class="nr-label" for="nr-c-cat">Category</label>
            <v-combobox id="nr-c-cat" v-model="meta.category" :items="categories" outlined dense hide-details clearable />
          </div>

          <div v-if="!usesMux" class="nr-field">
            <label class="nr-label" for="nr-c-max">Max connections</label>
            <v-text-field id="nr-c-max" v-model="data.max_connections" type="number" outlined dense hide-details="auto" class="nr-input-mono" :rules="[ rules.required, rules.nonZero, rules.onlyPositive ]" />
          </div>

          <div class="nr-field is-wide">
            <label class="nr-label" for="nr-c-desc">Description</label>
            <v-textarea id="nr-c-desc" v-model="meta.description" outlined dense hide-details rows="2" auto-grow maxlength="500" placeholder="What is on, who is it for" />
          </div>
        </div>

        <div class="nr-toggles">
          <div class="nr-toggle">
            <div><span>Control protection</span><small>Users can gain control only if at least one admin is in the room</small></div>
            <v-switch v-model="data.control_protection" inset hide-details aria-label="Control protection" />
          </div>
          <div class="nr-toggle">
            <div><span>Implicit control</span><small>Anyone can use the browser without asking for control first</small></div>
            <v-switch v-model="data.implicit_control" inset hide-details aria-label="Implicit control" />
          </div>
          <div v-if="storageEnabled && browserPolicyConfig" class="nr-toggle">
            <div>
              <span>Ad blocking &amp; browser policy</span>
              <small>{{ policyLine }} · <a href="#" @click.prevent="open.policy = true">customize</a></small>
            </div>
            <v-switch v-model="browserPolicyEnabled" inset hide-details aria-label="Ad blocking and browser policy" />
          </div>
        </div>

        <!-- advanced container specs, closed by default -->
        <section class="nr-adv">
          <div class="nr-adv-head">
            <span class="nr-label">Advanced container specs</span>
            <span class="nr-mono">defaults are fine</span>
          </div>

          <button type="button" class="nr-adv-row" :aria-expanded="open.av" @click="open.av = !open.av">
            <v-icon small>{{ open.av ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>Video &amp; audio
            <span class="nr-mono">{{ summaryAv }}</span>
          </button>
          <div v-if="open.av" class="nr-adv-body nr-grid2">
            <div class="nr-field">
              <label class="nr-label" for="nr-c-fps">Max fps</label>
              <v-text-field id="nr-c-fps" v-model="data.video_max_fps" outlined dense hide-details="auto" class="nr-input-mono" :disabled="!maxFpsEnabled" :rules="[ rules.required, rules.nonZero, rules.onlyPositive ]" />
            </div>
            <div class="nr-field nr-field-switch">
              <v-switch v-model="maxFpsEnabled" inset hide-details label="Cap the frame rate" />
            </div>
            <div class="nr-field">
              <span class="nr-label">Video codec</span>
              <v-select v-model="data.video_codec" :items="videoCodecs" outlined dense hide-details class="nr-input-mono" aria-label="Video codec" />
            </div>
            <div class="nr-field">
              <label class="nr-label" for="nr-c-vb">Video bitrate</label>
              <v-text-field id="nr-c-vb" v-model="data.video_bitrate" type="number" outlined dense hide-details="auto" class="nr-input-mono" :rules="[ rules.required, rules.nonZero, rules.onlyPositive ]" />
            </div>
            <div class="nr-field">
              <span class="nr-label">Audio codec</span>
              <v-select v-model="data.audio_codec" :items="audioCodecs" outlined dense hide-details class="nr-input-mono" aria-label="Audio codec" />
            </div>
            <div class="nr-field">
              <label class="nr-label" for="nr-c-ab">Audio bitrate</label>
              <v-text-field id="nr-c-ab" v-model="data.audio_bitrate" type="number" outlined dense hide-details="auto" class="nr-input-mono" :rules="[ rules.required, rules.nonZero, rules.onlyPositive ]" />
            </div>
          </div>

          <button type="button" class="nr-adv-row" :aria-expanded="open.res" @click="open.res = !open.res">
            <v-icon small>{{ open.res ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>Resources
            <span class="nr-mono">{{ summaryRes }}</span>
          </button>
          <div v-if="open.res && data.resources" class="nr-adv-body">
            <div class="nr-slider">
              <span>Memory</span>
              <v-slider v-model="data.resources.memory" :min="0" :max="8*1e9" :step="0.2*1e9" hide-details aria-label="Memory" />
              <span class="nr-mono">{{ data.resources.memory ? fmtMemory(data.resources.memory) : 'no limit' }}</span>
            </div>
            <div class="nr-slider">
              <span>CPUs</span>
              <v-slider v-model="data.resources.nano_cpus" :min="0" :max="8*1e9" :step="0.2*1e9" hide-details aria-label="CPUs" />
              <span class="nr-mono">{{ data.resources.nano_cpus ? (data.resources.nano_cpus / 1e9).toFixed(1) : 'no limit' }}</span>
            </div>
            <div class="nr-slider">
              <span>Shared memory</span>
              <v-slider v-model="data.resources.shm_size" :min="0" :max="20*1e9" :step="0.2*1e9" hide-details aria-label="Shared memory" />
              <span class="nr-mono">{{ data.resources.shm_size ? fmtMemory(data.resources.shm_size) : 'default' }}</span>
            </div>
            <div class="nr-toggle">
              <div><span>Nvidia GPU</span><small>Needs the Nvidia docker runtime</small></div>
              <v-switch
                :input-value="!!(data.resources.gpus && data.resources.gpus.length)"
                inset
                hide-details
                aria-label="Nvidia GPU"
                @change="$set(data.resources, 'gpus', $event ? ['all'] : [])"
              />
            </div>
            <div class="nr-field">
              <span class="nr-label">Devices</span>
              <v-combobox v-model="data.resources.devices" :items="[ '/dev/dri/renderD128' ]" multiple small-chips deletable-chips outlined dense hide-details class="nr-input-mono" aria-label="Devices" />
            </div>
          </div>

          <button type="button" class="nr-adv-row" :aria-expanded="open.env" @click="open.env = !open.env">
            <v-icon small>{{ open.env ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>Environment
            <span class="nr-mono">{{ envList.length }} {{ envList.length === 1 ? 'var' : 'vars' }}</span>
          </button>
          <div v-if="open.env" class="nr-adv-body">
            <div v-for="({ key, val }, index) in envList" :key="'env-' + index" class="nr-pair">
              <v-text-field :value="key" outlined dense hide-details class="nr-input-mono" placeholder="KEY" autocomplete="off" aria-label="Key" @input="setEnv(index, { key: $event, val })" />
              <span class="nr-mono">=</span>
              <v-text-field :value="val" outlined dense hide-details class="nr-input-mono" placeholder="value" autocomplete="off" aria-label="Value" @input="setEnv(index, { key, val: $event })" />
              <button type="button" class="nr-btn is-icon is-sm is-danger" aria-label="Remove variable" @click="delEnv(index)"><v-icon small>mdi-close</v-icon></button>
            </div>
            <button type="button" class="nr-btn is-sm is-secondary nr-add" @click="addEnv"><v-icon small>mdi-plus</v-icon>Add variable</button>
          </div>

          <button type="button" class="nr-adv-row" :aria-expanded="open.mounts" @click="open.mounts = !open.mounts">
            <v-icon small>{{ open.mounts ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>Mounts
            <span class="nr-mono">{{ (data.mounts || []).length || 'none' }}</span>
          </button>
          <div v-if="open.mounts" class="nr-adv-body">
            <div v-for="({ type, host_path, container_path }, index) in data.mounts" :key="'mount-' + index" class="nr-mount">
              <v-select :items="mountTypes" :value="type" outlined dense hide-details aria-label="Mount type" @input="setMount(index, { type: $event, host_path, container_path })" />
              <v-text-field :value="host_path" outlined dense hide-details="auto" class="nr-input-mono" placeholder="host path" autocomplete="off" aria-label="Host path" :rules="[ rules.absolutePath ]" @input="setMount(index, { type, host_path: $event, container_path })" />
              <v-text-field :value="container_path" outlined dense hide-details="auto" class="nr-input-mono" placeholder="container path" autocomplete="off" aria-label="Container path" :rules="[ rules.absolutePath ]" @input="setMount(index, { type, host_path, container_path: $event })" />
              <button type="button" class="nr-btn is-icon is-sm is-danger" aria-label="Remove mount" @click="delMount(index)"><v-icon small>mdi-close</v-icon></button>
            </div>
            <p v-if="data.mounts && data.mounts.length" class="nr-note">
              <span v-if="!storageEnabled">Private and Template mounts need storage, which is not enabled.<br></span>
              <span :class="{ 'is-dim': !storageEnabled }"><strong>Private</strong>: relative to <code>&lt;storage&gt;/rooms/&lt;room&gt;/</code>. <strong>Template</strong>: relative to <code>&lt;storage&gt;/templates/</code>, read only.</span><br>
              <strong>Protected</strong>: a whitelisted host path, read only. <strong>Public</strong>: a whitelisted host path.
            </p>
            <button type="button" class="nr-btn is-sm is-secondary nr-add" @click="addMount"><v-icon small>mdi-plus</v-icon>Add mount</button>
          </div>

          <button type="button" class="nr-adv-row" :aria-expanded="open.net" @click="open.net = !open.net">
            <v-icon small>{{ open.net ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>Network
            <span class="nr-mono">{{ data.hostname || 'container name' }}</span>
          </button>
          <div v-if="open.net" class="nr-adv-body nr-grid2">
            <div class="nr-field">
              <label class="nr-label" for="nr-c-host">Hostname</label>
              <v-text-field id="nr-c-host" v-model="data.hostname" outlined dense hide-details class="nr-input-mono" placeholder="container name" autocomplete="off" />
            </div>
            <div class="nr-field">
              <span class="nr-label">DNS</span>
              <v-combobox v-model="data.dns" multiple small-chips deletable-chips outlined dense hide-details class="nr-input-mono" placeholder="system default" aria-label="DNS" />
            </div>
          </div>

          <button type="button" class="nr-adv-row" :aria-expanded="open.pipes" @click="open.pipes = !open.pipes">
            <v-icon small>{{ open.pipes ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>Pipelines
            <span class="nr-mono">{{ summaryPipes }}</span>
          </button>
          <div v-if="open.pipes" class="nr-adv-body">
            <div class="nr-pipe">
              <v-switch v-model="videoPipelineEnabled" inset hide-details label="Video pipeline" />
              <v-textarea v-if="videoPipelineEnabled" v-model="data.video_pipeline" outlined dense hide-details rows="1" auto-grow class="nr-input-mono" aria-label="Video pipeline" />
            </div>
            <div class="nr-pipe">
              <v-switch v-model="audioPipelineEnabled" inset hide-details label="Audio pipeline" />
              <v-textarea v-if="audioPipelineEnabled" v-model="data.audio_pipeline" outlined dense hide-details rows="1" auto-grow class="nr-input-mono" aria-label="Audio pipeline" />
            </div>
            <div class="nr-pipe">
              <v-switch v-model="broadcastPipelineEnabled" inset hide-details label="Broadcast pipeline" />
              <v-textarea v-if="broadcastPipelineEnabled" v-model="data.broadcast_pipeline" outlined dense hide-details rows="1" auto-grow class="nr-input-mono" aria-label="Broadcast pipeline" />
            </div>
          </div>

          <button type="button" class="nr-adv-row" :aria-expanded="open.policy" @click="open.policy = !open.policy">
            <v-icon small>{{ open.policy ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>Browser policy
            <span class="nr-mono">{{ summaryPolicy }}</span>
          </button>
          <div v-if="open.policy" class="nr-adv-body">
            <p v-if="!storageEnabled" class="nr-note">Browser policy needs storage, which is not enabled.</p>
            <p v-else-if="!browserPolicyConfig" class="nr-note">Browser policy is not available for this image.</p>
            <template v-else>
              <div class="nr-toggle">
                <div><span>Use a browser policy</span><small>Extensions and browser settings for this room</small></div>
                <v-switch v-model="browserPolicyEnabled" inset hide-details aria-label="Use a browser policy" />
              </div>
              <div class="nr-field">
                <span class="nr-label">Extensions</span>
                <v-select v-model="browserPolicyContent.extensions" :items="browserPolicyExtensions" multiple small-chips outlined dense hide-details :disabled="!browserPolicyEnabled" aria-label="Extensions" />
              </div>
              <div class="nr-toggle">
                <div><span>Developer tools</span></div>
                <v-switch v-model="browserPolicyContent.developer_tools" inset hide-details :disabled="!browserPolicyEnabled" aria-label="Developer tools" />
              </div>
              <div class="nr-toggle">
                <div><span>Persistent data</span><small>Keep the browser profile between restarts</small></div>
                <v-switch v-model="browserPolicyContent.persistent_data" inset hide-details :disabled="!browserPolicyEnabled" aria-label="Persistent data" />
              </div>
            </template>
          </div>

          <button type="button" class="nr-adv-row" :aria-expanded="open.neko" @click="open.neko = !open.neko">
            <v-icon small>{{ open.neko ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>Neko options
            <span class="nr-mono">{{ nekoOnCount ? nekoOnCount + ' on' : 'defaults' }}</span>
          </button>
          <div v-if="open.neko" class="nr-adv-body">
            <v-text-field v-model="nekoSearch" outlined dense hide-details clearable prepend-inner-icon="mdi-magnify" :placeholder="'Search ' + nekoOptions.length + ' options'" aria-label="Search neko options" />
            <p class="nr-note">Every option of neko 3. Switch one on to set it for this room; variables under Environment win over these.</p>
            <div v-for="g in nekoGroups" :key="g.name" class="nr-opt-group">
              <span class="nr-label">{{ g.name }}</span>
              <div v-for="o in g.options" :key="o.env" class="nr-opt" :class="{ 'is-on': opt(o).on, 'is-managed': o.managed }">
                <div class="nr-opt-head">
                  <div class="nr-opt-text">
                    <span class="nr-mono nr-opt-env">{{ o.env }}</span>
                    <small>{{ o.description }}</small>
                  </div>
                  <span v-if="o.managed" class="nr-tag" title="neko-rooms sets this for every room">set by neko-rooms</span>
                  <v-switch v-else :input-value="opt(o).on" inset hide-details :aria-label="o.env" @change="setOpt(o, { on: !!$event })" />
                </div>
                <div v-if="opt(o).on" class="nr-opt-value">
                  <div v-if="o.type === 'boolean'" class="nr-seg is-field" role="group" :aria-label="o.env">
                    <button type="button" :class="{ 'is-on': opt(o).value === 'true' }" @click="setOpt(o, { value: 'true' })">true</button>
                    <button type="button" :class="{ 'is-on': opt(o).value === 'false' }" @click="setOpt(o, { value: 'false' })">false</button>
                  </div>
                  <v-text-field v-else :value="opt(o).value" outlined dense hide-details class="nr-input-mono" :placeholder="o.default || o.type" :aria-label="o.env" @input="setOpt(o, { value: $event || '' })" />
                  <span class="nr-mono nr-opt-default">{{ o.type }} · default {{ o.default || 'none' }}</span>
                </div>
              </div>
            </div>
            <p v-if="!nekoGroups.length" class="nr-note">No option matches.</p>
          </div>
        </section>
      </v-form>
    </div>

    <footer class="nr-create-foot">
      <span class="nr-mono nr-create-summary">{{ engineOf(data.neko_image) }} · {{ data.screen }} · {{ usesMux ? 'mux' : data.max_connections + ' seats' }}</span>
      <span class="flex-grow-1" />
      <button type="button" class="nr-btn" @click="Close">Cancel</button>
      <button type="button" class="nr-btn is-secondary" :disabled="loading" @click="Create(false)">
        <v-progress-circular v-if="loading && !loadingWithStart" indeterminate size="14" width="2" />Create
      </button>
      <button type="button" class="nr-btn is-primary" :disabled="loading" @click="CreateAndStart()">
        <v-progress-circular v-if="loadingWithStart" indeterminate size="14" width="2" />Create &amp; start
      </button>
    </footer>
  </v-card>
</template>

<style lang="scss" scoped>
.nr-create {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.nr-create-head {
  flex: none;
  display: flex;
  align-items: center;
  gap: 12px;
  height: 64px;
  padding: 0 16px 0 24px;
  border-bottom: 1px solid var(--border-subtle);
  h2 { margin: 0; font-size: 17px !important; font-weight: 600; letter-spacing: -0.025em !important; }
}
.nr-create-quota { font-size: 11.5px; color: var(--text-tertiary); }
.nr-create-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  > form { display: flex; flex-direction: column; gap: 24px; }
}
.nr-create-foot {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 72px;
  padding: 12px 24px;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-surface-1);
}
.nr-create-summary { font-size: 11.5px; color: var(--text-tertiary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; min-width: 0; }

.nr-templates { display: flex; flex-wrap: wrap; gap: 8px; }
.nr-template {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 30px;
  padding: 0 10px;
  border-radius: 7px;
  border: 1px solid var(--border-subtle);
  background: transparent;
  color: var(--text-secondary);
  font: 500 13px var(--font-sans);
  cursor: pointer;
  transition: border-color var(--dur) var(--ease), color var(--dur) var(--ease), background-color var(--dur) var(--ease);
  .v-icon { color: inherit !important; }
  &:hover { color: var(--text-primary); border-color: var(--border-hover); }
  &.is-on { color: var(--text-primary); background: var(--bg-surface-3); border-color: var(--border-hover); }
}

.nr-grid2 {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}
.nr-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  &.is-wide { grid-column: 1 / -1; }
}
.nr-field-switch { justify-content: flex-end; padding-bottom: 6px; }
.nr-hint { font-size: 11.5px; color: var(--text-tertiary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.nr-engines { display: grid; grid-template-columns: repeat(auto-fit, minmax(110px, 1fr)); gap: 8px; }
.nr-engine {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
  padding: 12px;
  border-radius: var(--radius-md);
  background: var(--bg-surface-2);
  border: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  text-align: left;
  cursor: pointer;
  transition: border-color var(--dur) var(--ease), background-color var(--dur) var(--ease), color var(--dur) var(--ease);
  &:hover { border-color: var(--border-hover); }
  &.is-on { border-color: var(--border-focus); background: var(--bg-surface-3); color: var(--text-primary); }
}
.nr-engine-name { font: 500 13px var(--font-sans); text-transform: capitalize; }
.nr-engine-image { max-width: 100%; font-size: 10.5px; color: var(--text-tertiary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.nr-toggles { display: flex; flex-direction: column; gap: 16px; }
.nr-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  > div:first-child { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
  span { font-size: 13.5px; color: var(--text-primary); }
  small { font-size: 12.5px; color: var(--text-tertiary); }
  .v-input { flex: none; margin: 0; padding: 0; }
}

.nr-adv {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  overflow: hidden;
}
.nr-adv-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  height: 44px;
  padding: 0 16px;
  background: var(--bg-surface-2);
  .nr-label { color: var(--text-secondary) !important; }
  .nr-mono { font-size: 11px; color: var(--text-tertiary); }
}
.nr-adv-row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  height: 44px;
  padding: 0 16px;
  border: 0;
  border-top: 1px solid var(--border-subtle);
  background: transparent;
  color: var(--text-primary);
  font: 500 13.5px var(--font-sans);
  text-align: left;
  cursor: pointer;
  transition: background-color var(--dur) var(--ease);
  &:hover { background: rgba(255, 255, 255, .02); }
  .v-icon { color: var(--text-tertiary) !important; }
  .nr-mono { margin-left: auto; font-size: 11.5px; color: var(--text-tertiary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 60%; }
}
.nr-adv-body {
  padding: 8px 16px 20px 40px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  &.nr-grid2 { display: grid; }
}
.nr-slider {
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr) 72px;
  align-items: center;
  gap: 16px;
  > span:first-child { font-size: 13px; color: var(--text-secondary); }
  > .nr-mono { font-size: 12.5px; text-align: right; color: var(--text-primary); }
  .v-input { margin: 0; padding: 0; }
}
.nr-pair, .nr-mount {
  display: grid;
  align-items: center;
  gap: 8px;
}
.nr-pair { grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr) 30px; > .nr-mono { color: var(--text-tertiary); } }
.nr-mount { grid-template-columns: 112px minmax(0, 1fr) minmax(0, 1fr) 30px; }
.nr-add { align-self: flex-start; }
.nr-pipe { display: flex; flex-direction: column; gap: 8px; .v-input--switch { margin: 0; } }
.nr-opt-group { display: flex; flex-direction: column; gap: 8px; }
.nr-opt {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  transition: border-color var(--dur) var(--ease), background-color var(--dur) var(--ease);
  &.is-on { border-color: var(--border-hover); background: var(--bg-surface-2); }
  &.is-managed { opacity: .55; }
}
.nr-opt-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  .v-input { flex: none; margin: 0; padding: 0; }
}
.nr-opt-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  small { font-size: 12px; line-height: 1.45; color: var(--text-tertiary); }
}
.nr-opt-env { font-size: 11.5px; color: var(--text-primary); word-break: break-all; }
.nr-opt-value { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; > .v-input { flex: 1; min-width: 160px; } }
.nr-opt-default { font-size: 11px; color: var(--text-tertiary); }
.nr-note {
  margin: 0;
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--text-tertiary);
  strong { color: var(--text-secondary); font-weight: 500; }
  .is-dim { opacity: .5; }
}

@media (max-width: 600px) {
  .nr-grid2 { grid-template-columns: minmax(0, 1fr); }
  .nr-adv-body { padding-left: 16px; }
  .nr-slider { grid-template-columns: 96px minmax(0, 1fr) 64px; }
  .nr-mount { grid-template-columns: minmax(0, 1fr) 30px; }
  .nr-create-summary { display: none; }
  .nr-create-foot { flex-wrap: wrap; }
}
</style>

<style lang="scss">
// monospace for values machines read: names, passwords, paths, codecs
.nr-input-mono input, .nr-input-mono textarea, .nr-input-mono .v-select__selection {
  font-family: var(--font-mono) !important;
  font-size: 13px !important;
}
.nr-create .v-text-field--outlined .v-input__append-inner { margin-top: 0 !important; align-self: center; display: flex; gap: 2px; margin-right: -8px; }
</style>

<script lang="ts">
import { Vue, Component, Ref, Watch } from 'vue-property-decorator'
import { AxiosError } from 'axios'
import { randomPassword } from '@/utils/random'
import { copyText } from '@/utils/join'
import { NEKO_OPTIONS, NekoOption } from '@/utils/nekoEnv'

interface OptState {
  on: boolean
  value: string
}

import {
  RoomSettings,
  BrowserPolicyContent,
  BrowserPolicyExtension,
  BrowserPolicyTypeEnum,
  RoomMount,
  RoomMountTypeEnum,
} from '@/api/index'

// room templates fill in the form for common kinds of rooms
const TEMPLATES = [
  { key: 'blank', name: 'Blank', note: 'An empty browser', icon: 'mdi-web', title: '', category: '', url: '', extensions: [] as string[] },
  { key: 'youtube', name: 'YouTube', note: 'Ad block & SponsorBlock', icon: 'mdi-youtube', title: 'YouTube together', category: 'Movies', url: 'https://www.youtube.com', extensions: ['ublock', 'sponsorblock'] },
  { key: 'twitch', name: 'Twitch', note: 'Watch streams with ad block', icon: 'mdi-twitch', title: 'Twitch watch party', category: 'Gaming', url: 'https://www.twitch.tv', extensions: ['ublock'] },
  { key: 'music', name: 'Music', note: 'Listen together', icon: 'mdi-music', title: 'Listening party', category: 'Music', url: 'https://music.youtube.com', extensions: ['ublock'] },
  { key: 'movie', name: 'Movie night', note: 'Plan it as a watch party', icon: 'mdi-movie-open-outline', title: 'Movie night', category: 'Movies', url: '', extensions: ['ublock'] },
]

export type VForm = Vue & {
  validate: () => boolean;
  resetValidation: () => boolean;
  reset: () => void;
}

@Component
export default class RoomsCreate extends Vue {
  @Ref('form') readonly _form!: VForm

  public valid = true

  public meta = {
    title: '',
    description: '',
    category: '' as string | null,
    visibility: 'private',
    // eslint-disable-next-line
    start_url: '' as string | null,
  }

  public template = 'blank'

  get templates() {
    return TEMPLATES
  }

  // useTemplate fills the form for a kind of room
  useTemplate(key: string) {
    const t = TEMPLATES.find(t => t.key === key)
    if (!t) return
    const previous = TEMPLATES.find(p => p.key === this.template)
    this.template = key

    // only replace what the previous template filled in
    if (!this.meta.title || (previous && this.meta.title === previous.title)) this.meta.title = t.title
    if (!this.meta.category || (previous && this.meta.category === previous.category)) this.meta.category = t.category
    // eslint-disable-next-line
    this.meta.start_url = t.url

    // ad blocking and friends, where browser policies work
    if (this.storageEnabled && this.browserPolicyConfig) {
      const wanted = this.browserPolicyExtensions
        .filter((e: { text: string } | undefined) => e && t.extensions.some(name => e.text.toLowerCase().includes(name)))
        .map((e: { value: BrowserPolicyExtension }) => e.value)
      this.browserPolicyEnabled = wanted.length > 0
      this.browserPolicyContent = { ...this.browserPolicyContent, extensions: wanted }
    }
  }

  get categories(): string[] {
    return this.$store.state.app.branding.home.categories || []
  }

  get visibilityItems() {
    const policy = this.$store.getters.policy
    const isAdmin = this.$store.getters.isAdmin
    return [
      { text: 'Public, listed on the homepage', short: 'Public', value: 'public', icon: 'mdi-earth', disabled: !isAdmin && !(policy && policy.users_can_make_public) },
      { text: 'Friends only', short: 'Friends', value: 'friends', icon: 'mdi-account-multiple', disabled: !(policy && policy.friends_enabled) },
      { text: 'Private, invite link only', short: 'Invite', value: 'private', icon: 'mdi-lock' },
    ]
  }

  // which advanced sections are open
  public open = { av: false, res: false, env: false, mounts: false, net: false, pipes: false, policy: false, neko: false }

  // neko options switched on for this room, by environment variable
  public nekoOpts: Record<string, OptState> = {}
  public nekoSearch = ''

  get nekoOptions(): NekoOption[] {
    return NEKO_OPTIONS
  }

  get nekoGroups(): { name: string; options: NekoOption[] }[] {
    const q = (this.nekoSearch || '').trim().toLowerCase()
    const groups: { name: string; options: NekoOption[] }[] = []
    NEKO_OPTIONS
      .filter(o => !q || o.env.toLowerCase().includes(q) || o.description.toLowerCase().includes(q))
      .forEach(o => {
        let g = groups.find(x => x.name === o.group)
        if (!g) groups.push(g = { name: o.group, options: [] })
        g.options.push(o)
      })
    return groups
  }

  get nekoOnCount(): number {
    return Object.values(this.nekoOpts).filter(v => v.on).length
  }

  opt(o: NekoOption): OptState {
    return this.nekoOpts[o.env] || { on: false, value: '' }
  }

  // switching an option on starts from a useful value: the opposite of the
  // default for a switch, the default for anything else
  setOpt(o: NekoOption, patch: Partial<OptState>) {
    const cur = this.opt(o)
    const next = { ...cur, ...patch }
    if (patch.on && !cur.on && next.value === '') {
      next.value = o.type === 'boolean' ? (o.default === 'true' ? 'false' : 'true') : o.default
    }
    Vue.set(this.nekoOpts, o.env, next)
  }

  // the extensions a room gets without anyone picking them: ad blocking
  get recommendedExtensions() {
    return this.browserPolicyExtensions
      .filter((e: { text: string } | undefined) => e && e.text.toLowerCase().includes('ublock'))
      .map((e: { value: BrowserPolicyExtension }) => e.value)
  }

  get policyLine(): string {
    if (!this.browserPolicyEnabled) return 'Off'
    const picked = this.browserPolicyContent.extensions || []
    const names = this.browserPolicyExtensions
      .filter((e: { text: string; value: BrowserPolicyExtension } | undefined) => e && picked.some(p => JSON.stringify(p) === JSON.stringify(e.value)))
      .map((e: { text: string }) => e.text)
    return names.length ? names.join(', ') : 'On, no extensions'
  }

  // a browser policy is on by default, with ad blocking, wherever the image
  // and the server support one
  @Watch('data.neko_image')
  applyPolicyDefaults() {
    if (!this.storageEnabled || !this.browserPolicyConfig) {
      this.browserPolicyEnabled = false
      return
    }
    this.browserPolicyEnabled = true
    this.browserPolicyContent = { ...this.browserPolicyContent, extensions: this.recommendedExtensions }
  }
  public copied = ''
  private copiedTimer = 0

  public showUserPass = false
  public showAdminPass = false

  public maxFpsEnabled = true
  public videoPipelineEnabled = false
  public audioPipelineEnabled = false
  public broadcastPipelineEnabled = false
  public browserPolicyEnabled = false

  public loading = false
  public loadingWithStart = false
  public data: RoomSettings = { ...this.$store.state.defaultRoomSettings }
  public browserPolicyContent: BrowserPolicyContent = { ...this.$store.state.defaultBrowserPolicyContent }
  public envList: { key: string; val: string }[] = []

  // eslint-disable-next-line
  public rules: any = {
    // eslint-disable-next-line
    required(val: any) {
      return val === null || typeof val === 'undefined' || val === "" ? 'This filed is mandatory.' : true
    },
    minLen: (min: number) =>
      (val: string) => 
        val ? (val.length >= min || 'This field must have atleast ' + min + ' characters') : true,
    onlyPositive(val: number) {
      return val < 0 ? 'Value cannot be negative.' : true
    },
    nonZero(val: string) {
      return val === "0" ? 'Value cannot be zero.' : true
    },
    containerName(val: string) {
      return val && !/^[a-zA-Z0-9_.-]+$/.test(val) ? 'Must only contain a-z A-Z 0-9 _ . -' : true
    },
    containerNameStart(val: string) {
      return val && /^[_.-]/.test(val) ? 'Cannot start with _ . -' : true
    },
    absolutePath(val: string) {
      return val[0] !== "/" ? 'Must be absolute path, starting with /.' : true
    }
  }

  get host(): string {
    return window.location.host
  }

  get roomLimit(): number {
    const st = this.$store.state.app.status
    return st ? st.room_limit : 0
  }

  get roomCount(): number {
    const st = this.$store.state.app.status
    return st ? st.room_count : 0
  }

  get summaryAv(): string {
    const fps = this.maxFpsEnabled ? this.data.video_max_fps + ' fps' : 'no fps cap'
    return [fps, this.data.video_codec, this.data.audio_codec].filter(Boolean).join(' · ')
  }

  get summaryRes(): string {
    const r = this.data.resources
    if (!r) return 'no limits'
    const parts = [
      r.memory ? this.fmtMemory(r.memory) : 'no memory limit',
      r.nano_cpus ? (r.nano_cpus / 1e9).toFixed(1) + ' CPU' : 'no CPU limit',
    ]
    if (r.shm_size) parts.push(this.fmtMemory(r.shm_size) + ' shm')
    if (r.gpus && r.gpus.length) parts.push('GPU')
    return parts.join(' · ')
  }

  get summaryPipes(): string {
    const n = [this.videoPipelineEnabled, this.audioPipelineEnabled, this.broadcastPipelineEnabled].filter(Boolean).length
    return n ? n + ' custom' : 'defaults'
  }

  get summaryPolicy(): string {
    if (!this.storageEnabled || !this.browserPolicyConfig) return 'not available'
    if (!this.browserPolicyEnabled) return 'off'
    const n = (this.browserPolicyContent.extensions || []).length
    return n + (n === 1 ? ' extension' : ' extensions')
  }

  fmtMemory(value: number): string {
    if (value < 1e9) return (value / 1e6).toFixed(0) + ' MB'
    return (value / 1e9).toFixed(1) + ' GB'
  }

  // engineOf names the browser of an image, ghcr.io/m1k1o/neko/firefox:latest
  // and m1k1o/neko:firefox both run firefox
  engineOf(image?: string): string {
    const last = (image || '').split('/').pop() || ''
    const [base, tag] = last.split(':')
    return base === 'neko' && tag ? tag : (base || '')
  }

  generate(field: 'user_pass' | 'admin_pass') {
    Vue.set(this.data, field, randomPassword())
    if (field === 'user_pass') this.showUserPass = true
    else this.showAdminPass = true
  }

  async copy(field: 'user_pass' | 'admin_pass') {
    const value = this.data[field]
    if (!value) return
    if (await copyText(value)) {
      this.copied = field
      window.clearTimeout(this.copiedTimer)
      this.copiedTimer = window.setTimeout(() => { this.copied = '' }, 1500)
    }
  }

  get nekoImages() {
    return this.$store.state.roomsConfig.neko_images
  }

  get storageEnabled() {
    return this.$store.state.roomsConfig.storage_enabled
  }

  get usesMux() {
    return this.$store.state.roomsConfig.uses_mux
  }

  get videoCodecs() {
    return this.$store.state.videoCodecs
  }

  get audioCodecs() {
    return this.$store.state.audioCodecs
  }

  get availableScreens() {
    return this.$store.state.availableScreens
  }

  get mountTypes() {
    return [
      ...(this.storageEnabled ? [
        {
          text: 'Private',
          value: 'private',
        },
        {
          text: 'Template',
          value: 'template',
        },
      ] : []),
      {
        text: 'Protected',
        value: 'protected',
      },
      {
        text: 'Public',
        value: 'public',
      },
    ]
  }

  get browserPolicyConfig() {
    const nekoImage = this.data.neko_image
    if (!nekoImage) return undefined

    return this.$store.state.browserPolicyConfig.find(
      // find browser policy config for this image
      ({ images }: { images: string[] }): boolean => images.some(
        // if nekoImage includes the image name
        (image: string): boolean => nekoImage.includes(image)
      )
    )
  }

  get browserPolicyExtensions() {
    const config = this.browserPolicyConfig
    if (!config) return []
  
    return this.$store.state.browserPolicyExtensions.map(({ text, value: values }: {
      text: string;
      value: Record<BrowserPolicyTypeEnum, BrowserPolicyExtension>;
    }) => {
      const value = values[config.type as BrowserPolicyTypeEnum]
      if (!value) return undefined
      return { text, value }
    })
  }

  @Watch('browserPolicyContent.persistent_data')
  onPersistentDataUpdate(enabled: boolean) {
    const config = this.browserPolicyConfig
    if (!config) return

    if (enabled) {
      // eslint-disable-next-line
      this.data.mounts = [ ...(this.data.mounts || []), { type: RoomMountTypeEnum.private, host_path: '/profile', container_path: config.profile }]
    } else {
      // eslint-disable-next-line
      this.data.mounts = (this.data.mounts || []).filter(({ type, container_path }) => type != RoomMountTypeEnum.private && container_path != config.profile)
    }
  }

  addEnv() {
    Vue.set(this, 'envList', [ ...this.envList, { key: '', val: '' } ])
  }

  setEnv(index: number, data: { key: string; val: string }) {
    Vue.set(this.envList, index, data)
  }

  delEnv(index: number) {
    Vue.delete(this.envList, index)
  }

  addMount() {
    const type = this.storageEnabled ? RoomMountTypeEnum.private : RoomMountTypeEnum.public
    Vue.set(this.data, 'mounts', [ ...(this.data.mounts || []), { type, host_path: '', container_path: '' } ])
  }

  setMount(index: number, data: RoomMount) {
    let mounts = this.data.mounts || []
    if (index >= mounts.length) {
      mounts = [ ...mounts, data ]
    } else {
      mounts[index] = data
    }
    Vue.set(this.data, 'mounts', [ ...mounts])
  }

  delMount(index: number) {
    const mounts = this.data.mounts || []
    mounts.splice(index, 1)
    Vue.set(this.data, 'mounts', [ ...mounts])
  }

  async CreateAndStart() {
    this.loadingWithStart = true
    try {
      await this.Create(true)
    } finally {
      this.loadingWithStart = false
    }
  }

  async Create(start = false) {
    const valid = this._form.validate()
    if (!valid) return

    this.loading = true

    try {
      // neko options first, the custom variables win over them
      const options = Object.entries(this.nekoOpts)
        .filter(([, v]) => v.on && v.value !== '')
        .reduce((obj, [key, v]) => ({ ...obj, [key]: v.value }), {} as Record<string, string>)
      const envs = this.envList.reduce((obj, { key, val }) => ({ ...obj, [key]: val, }), options)

      await this.$store.dispatch(start ? 'ROOMS_CREATE_AND_START' : 'ROOMS_CREATE', {
        ...this.data,
        // eslint-disable-next-line
        user_pass: this.data.user_pass || randomPassword(),
        // eslint-disable-next-line
        admin_pass: this.data.admin_pass || randomPassword(),
        // eslint-disable-next-line
        max_connections: Number(this.data.max_connections), // ignored when uses mux
        // eslint-disable-next-line
        control_protection: Boolean(this.data.control_protection),
        // eslint-disable-next-line
        video_bitrate: Number(this.data.video_bitrate),
        // eslint-disable-next-line
        video_pipeline: this.videoPipelineEnabled ? this.data.video_pipeline : '',
        // eslint-disable-next-line
        video_max_fps: this.maxFpsEnabled ? Number(this.data.video_max_fps) : 0,
        // eslint-disable-next-line
        audio_bitrate: Number(this.data.audio_bitrate),
        // eslint-disable-next-line
        audio_pipeline: this.audioPipelineEnabled ? this.data.audio_pipeline : '',
        // eslint-disable-next-line
        broadcast_pipeline: this.broadcastPipelineEnabled ? this.data.broadcast_pipeline : '',
        envs,
        // eslint-disable-next-line
        browser_policy: this.browserPolicyEnabled && this.browserPolicyConfig ? {
          type: this.browserPolicyConfig.type,
          path: this.browserPolicyConfig.path,
          content: this.browserPolicyContent
        } : undefined,
        // eslint-disable-next-line
        meta: { ...this.meta, category: this.meta.category || '', start_url: this.meta.start_url || '' },
      })
      this.Clear()
      this.$emit('finished', true)
     } catch(e) {
      const response = (e as AxiosError).response
      if (response) {
        this.$swal({
          title: 'Server error',
          text: String(response.data),
          icon: 'error',
        })
      } else {
        this.$swal({
          title: 'Network error',
          text: String(e),
          icon: 'error',
        })
      }
    } finally {
      this.loading = false
    }
  }

  Clear() {
    this._form.resetValidation()
    // eslint-disable-next-line
    this.meta = { title: '', description: '', category: '', visibility: 'private', start_url: '' }
    this.template = 'blank'
    this.data = {
      ...this.$store.state.defaultRoomSettings,
      // eslint-disable-next-line
      neko_image: this.nekoImages[0],
    }
    this.browserPolicyContent = { ...this.$store.state.defaultBrowserPolicyContent }
    this.envList = Object.entries({...this.data.envs}).map(([ key, val ]) => ({ key, val, }))
    this.nekoOpts = {}
    this.nekoSearch = ''
    this.applyPolicyDefaults()
  }

  Close() {
    this.Clear()
    this.$emit('finished', true)
  }

  mounted() {
    this.Clear()
  }
}
</script>
