import { Component } from '@angular/core';
import { ProyectoService } from 'src/app/services/proyecto.service';
import Swal from 'sweetalert2';
import { Router } from '@angular/router';

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.css']
})
export class LoginComponent {

  idparticion = "";
  user = "";
  pwd = "";
  retorno = "";

  constructor(private service: ProyectoService, private router : Router) { }

  ngOnInit(): void {
  }

  ingresar() {
    if (!this.idparticion || !this.user || !this.pwd) {
      Swal.fire({
        title: 'Error',
        text: 'Debe llenar todos los campos',
        icon: 'error',
      });
      return;
    }
    console.log("ID Particion: ", this.idparticion);
    console.log("User: ", this.user);
    console.log("Password: ", this.pwd)

    //let avr = this.idparticion.trim()

    this.service.postLogin(this.idparticion, this.user, this.pwd).subscribe(async (res: any) => {
      let mensaje = res.result_log
      console.log(mensaje);

      switch (mensaje) {
        case "OK":
          Swal.fire({
            title: ':D',
            text: 'Bienvenido de nuevo ' + this.user,
            imageUrl: 'https://unsplash.it/400/200',
            imageWidth: 400,
            imageHeight: 200,
            imageAlt: 'Custom image',
          })
          this.router.navigate(['/usuario'], { queryParams: { idparticion: this.idparticion, user: this.user } });
          break
        case "NO":
          //console.log("Entra en el segundo");
          Swal.fire({
            icon: 'error',
            title: 'Oops... Algo salió mal',
            text: 'ID/Usuario/contraseña no encontrado',
            footer: '<a>Revisa tus credenciales correctamente</a>'
          })
          break
          
        default:
          console.log("No es correcto");
          break
          

      }
    });
  }


  


}
