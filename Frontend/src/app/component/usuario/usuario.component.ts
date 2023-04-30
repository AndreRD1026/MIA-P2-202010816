import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { ActivatedRoute } from '@angular/router';
import Swal from 'sweetalert2';
import { ProyectoService } from 'src/app/services/proyecto.service';

@Component({
  selector: 'app-usuario',
  templateUrl: './usuario.component.html',
  styleUrls: ['./usuario.component.css']
})
export class UsuarioComponent {

  idparticion: string;
  user: string;
  rutaa : string;
  pasarrepuesta : string;


  constructor(private service: ProyectoService, private router : Router, private route: ActivatedRoute) { 
    this.idparticion = ""
    this.user = ""
    this.rutaa = ""
    this.pasarrepuesta = ""
  }

  ngOnInit(): void {
    this.route.queryParams.subscribe(params => {
      this.idparticion = params['idparticion'];
      this.user = params['user'];
    });
  }


  cerrarSesion(){
    this.service.postLogout().subscribe(async (res: any) => {
      this.router.navigate(['/login'])
    });
  }

  generarDisk(){
    Swal.fire({
      icon: 'success',
      title: 'Abriendo Reporte',
      showConfirmButton: false,
      timer: 1500
    })
    this.service.postRepDisk(this.idparticion, this.rutaa).subscribe(async (res: any) => {
      this.pasarrepuesta = res.result_disk
      console.log(this.pasarrepuesta);
      this.router.navigate(['/reportes'], { queryParams: { pasarrepuesta: this.pasarrepuesta } });
    });
  }

  generarSB(){
    Swal.fire({
      icon: 'success',
      title: 'Abriendo Reporte',
      showConfirmButton: false,
      timer: 1500
    })
    this.service.postRepSB(this.idparticion, this.rutaa).subscribe(async (res: any) => {
      this.pasarrepuesta = res.result_sb
      console.log(this.pasarrepuesta);
      this.router.navigate(['/reportes'], { queryParams: { pasarrepuesta: this.pasarrepuesta } });
    });
  }

  generarTree(){
    Swal.fire({
      icon: 'success',
      title: 'Abriendo Reporte',
      showConfirmButton: false,
      timer: 1500
    })
    this.service.postRepTree(this.idparticion, this.rutaa).subscribe(async (res: any) => {
      this.pasarrepuesta = res.result_tree
      console.log(this.pasarrepuesta);
      this.router.navigate(['/reportes'], { queryParams: { pasarrepuesta: this.pasarrepuesta } });
    });
  }

  generarFile(){
    Swal.fire(
      'Lo siento',
      'Este reporte no está disponible :( ',
    )
  }

}