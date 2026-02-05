<script setup>
import { ref, onMounted, computed } from 'vue'
import { toast } from 'vue-sonner'
import api from '@/api'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import { ShoppingCart, CheckCircle, AlertCircle, RefreshCw } from 'lucide-vue-next'

const loading = ref(true)
const saving = ref(false)
const testing = ref(false)

// Form state
const providerType = ref('')
const baseURL = ref('')
const clientID = ref('')
const clientSecret = ref('')

// Status tracking
const hasConfig = ref(false)
const testStatus = ref(null) // 'success', 'error', or null

const providerName = computed(() => {
  switch (providerType.value) {
    case 'magento1': return 'Magento 1 / Maho Commerce'
    case 'magento2': return 'Magento 2'
    case 'shopify': return 'Shopify'
    default: return 'Ecommerce'
  }
})

onMounted(async () => {
  await fetchSettings()
})

async function fetchSettings() {
  loading.value = true
  try {
    const res = await api.getEcommerceSettings()
    if (res.data?.type) {
      providerType.value = res.data.type
      baseURL.value = res.data.base_url || ''
      clientID.value = res.data.client_id || ''
      hasConfig.value = true
    }
  } catch (err) {
    // Settings not configured yet
    console.log('Ecommerce settings not configured')
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  if (providerType.value && !baseURL.value) {
    toast.error('Please enter a Base URL')
    return
  }

  saving.value = true
  try {
    await api.updateEcommerceSettings({
      type: providerType.value,
      base_url: baseURL.value,
      client_id: clientID.value,
      client_secret: clientSecret.value
    })
    toast.success('Ecommerce settings saved')
    clientSecret.value = ''
    hasConfig.value = !!providerType.value
    testStatus.value = null
  } catch (err) {
    toast.error(err.response?.data?.message || 'Failed to save settings')
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  testing.value = true
  testStatus.value = null
  try {
    await api.testEcommerceConnection({
      type: providerType.value,
      base_url: baseURL.value,
      client_id: clientID.value,
      client_secret: clientSecret.value
    })
    toast.success('Connection successful!')
    testStatus.value = 'success'
  } catch (err) {
    toast.error(err.response?.data?.message || 'Connection failed')
    testStatus.value = 'error'
  } finally {
    testing.value = false
  }
}

function clearSettings() {
  providerType.value = ''
  baseURL.value = ''
  clientID.value = ''
  clientSecret.value = ''
  testStatus.value = null
}
</script>

<template>
  <AdminPageWithHelp>
    <template #content>
      <div v-if="loading" class="flex justify-center py-12">
        <Spinner />
      </div>

      <div v-else class="space-y-6">
        <Card>
          <CardHeader>
            <div class="flex items-center gap-2">
              <ShoppingCart class="h-5 w-5" />
              <CardTitle>Ecommerce Integration</CardTitle>
              <Badge v-if="hasConfig && providerType" variant="secondary">
                {{ providerName }}
              </Badge>
              <Badge v-if="testStatus === 'success'" variant="default" class="bg-green-500">
                <CheckCircle class="h-3 w-3 mr-1" />
                Connected
              </Badge>
            </div>
            <CardDescription>
              Connect your ecommerce platform to enable order lookups in AI responses.
              When customers ask about their orders, the AI assistant can fetch real-time order data.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <div class="space-y-2">
              <Label for="provider-type">Provider</Label>
              <Select v-model="providerType">
                <SelectTrigger>
                  <SelectValue placeholder="Select a provider" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">Disabled</SelectItem>
                  <SelectItem value="magento1">Magento 1 / Maho Commerce</SelectItem>
                  <SelectItem value="magento2">Magento 2</SelectItem>
                  <SelectItem value="shopify">Shopify</SelectItem>
                </SelectContent>
              </Select>
              <p class="text-xs text-muted-foreground">
                Select your ecommerce platform. Choose "Disabled" to turn off ecommerce integration.
              </p>
            </div>

            <template v-if="providerType">
              <div class="space-y-2">
                <Label for="base-url">Base URL</Label>
                <Input
                  id="base-url"
                  v-model="baseURL"
                  :placeholder="providerType === 'shopify' ? 'https://your-store.myshopify.com' : 'https://your-store.com'"
                />
                <p class="text-xs text-muted-foreground">
                  <template v-if="providerType === 'shopify'">
                    Your Shopify store URL (e.g., https://your-store.myshopify.com)
                  </template>
                  <template v-else-if="providerType === 'magento1'">
                    Your Magento 1 / Maho Commerce store URL with API endpoint (e.g., https://store.com/api/rest)
                  </template>
                  <template v-else>
                    Your Magento 2 store URL (e.g., https://store.com)
                  </template>
                </p>
              </div>

              <div class="space-y-2">
                <Label for="client-id">
                  <template v-if="providerType === 'shopify'">API Key</template>
                  <template v-else-if="providerType === 'magento1'">Consumer Key</template>
                  <template v-else>Integration Access Token</template>
                </Label>
                <Input
                  id="client-id"
                  v-model="clientID"
                  :placeholder="hasConfig ? '********' : 'Enter API key/token'"
                />
                <p class="text-xs text-muted-foreground">
                  <template v-if="providerType === 'shopify'">
                    Your Shopify Admin API access token
                  </template>
                  <template v-else-if="providerType === 'magento1'">
                    OAuth Consumer Key from System > Web Services > REST OAuth Consumers
                  </template>
                  <template v-else>
                    Integration Access Token from System > Extensions > Integrations
                  </template>
                </p>
              </div>

              <div class="space-y-2">
                <Label for="client-secret">
                  <template v-if="providerType === 'shopify'">Admin API Secret Key</template>
                  <template v-else-if="providerType === 'magento1'">Consumer Secret</template>
                  <template v-else>Integration Secret</template>
                </Label>
                <Input
                  id="client-secret"
                  v-model="clientSecret"
                  type="password"
                  :placeholder="hasConfig ? 'Enter new secret to change' : 'Enter secret'"
                />
                <p class="text-xs text-muted-foreground">
                  <template v-if="providerType === 'shopify'">
                    Your Shopify Admin API secret key (stored encrypted)
                  </template>
                  <template v-else-if="providerType === 'magento1'">
                    OAuth Consumer Secret (stored encrypted)
                  </template>
                  <template v-else>
                    Integration secret token (stored encrypted)
                  </template>
                </p>
              </div>
            </template>

            <div class="flex gap-2 pt-4">
              <Button @click="saveSettings" :disabled="saving">
                {{ saving ? 'Saving...' : 'Save' }}
              </Button>
              <Button
                v-if="providerType"
                variant="outline"
                @click="testConnection"
                :disabled="testing || !baseURL"
              >
                <RefreshCw v-if="testing" class="h-4 w-4 mr-2 animate-spin" />
                <CheckCircle v-else-if="testStatus === 'success'" class="h-4 w-4 mr-2 text-green-500" />
                <AlertCircle v-else-if="testStatus === 'error'" class="h-4 w-4 mr-2 text-red-500" />
                Test Connection
              </Button>
              <Button
                v-if="hasConfig && providerType"
                variant="destructive"
                @click="clearSettings"
              >
                Disable Integration
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>
    <template #help>
      <h4 class="font-medium mb-2">Ecommerce Integration</h4>
      <p class="text-sm text-muted-foreground mb-4">
        Connect your ecommerce platform to enhance AI responses with real-time order data.
      </p>
      <h4 class="font-medium mb-2">How It Works</h4>
      <p class="text-sm text-muted-foreground mb-4">
        When generating AI responses, the system can look up customer orders to provide
        accurate shipping status, order details, and product information.
      </p>
      <h4 class="font-medium mb-2">Supported Platforms</h4>
      <ul class="text-sm text-muted-foreground list-disc list-inside space-y-1">
        <li><strong>Magento 1 / Maho Commerce</strong> - REST API with OAuth 1.0</li>
        <li><strong>Magento 2</strong> - REST API with Integration tokens</li>
        <li><strong>Shopify</strong> - Admin API with access tokens</li>
      </ul>
    </template>
  </AdminPageWithHelp>
</template>
