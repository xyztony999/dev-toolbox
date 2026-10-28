import type {
  Alert,
  Button,
  Divider,
  Empty,
  Input,
  Layout,
  LayoutContent,
  LayoutHeader,
  LayoutSider,
  Menu,
  MenuItem,
  Space,
  Tag,
} from 'ant-design-vue'

declare module 'vue' {
  export interface GlobalComponents {
    'a-alert': typeof Alert
    'a-button': typeof Button
    'a-divider': typeof Divider
    'a-empty': typeof Empty
    'a-input': typeof Input
    'a-layout': typeof Layout
    'a-layout-content': typeof LayoutContent
    'a-layout-header': typeof LayoutHeader
    'a-layout-sider': typeof LayoutSider
    'a-menu': typeof Menu
    'a-menu-item': typeof MenuItem
    'a-space': typeof Space
    'a-tag': typeof Tag
  }
}

export {}
