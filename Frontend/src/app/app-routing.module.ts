import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { InicioComponent } from './component/inicio/inicio.component';
import { LoginComponent } from './component/login/login.component';
import { UsuarioComponent } from './component/usuario/usuario.component';

const routes: Routes = [
  {
    path: 'inicio',
    component: InicioComponent
  },{
    path: 'login',
    component : LoginComponent
  },{
    path : 'usuario',
    component : UsuarioComponent
  },{
    path: "**",
    redirectTo: 'inicio'
  }
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
