ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20024  10.155.84.3 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20025  10.155.84.3 &
ssh css-g88-002 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20026  10.155.84.3 &
ssh css-g88-002 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20027  10.155.84.3 &
ssh css-g88-004 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20028  10.155.84.3 &
ssh css-g88-004 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20029  10.155.84.3 &
